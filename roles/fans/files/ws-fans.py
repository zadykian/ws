#!/usr/bin/python3
# Written by github.com/zadykian/ws, whose next run replaces any change made here.
"""ws-fans: the board's fans on the curves of a config the fans role writes.

Each tick it reads the temperatures, moves each curve past its hysteresis, and steps each fan
towards its speed. A fan whose temperature or device is missing goes back to the board's own
curve, as every fan does when the service stops. docs/fans.md has the details.
"""

import argparse
import itertools
import json
import signal
import sys
import time
from collections import deque
from pathlib import Path

from fan_devices import PERCENT_FULL, HwmonFan, MissingError, Nvml, hwmon_temperature

WAYS = ("up", "down")


def log(message):
    """Write message to stderr, which systemd sends to the journal."""
    print(message, file=sys.stderr, flush=True)


def report(seen, key, problem):
    """Log key's problem where it differs from the last one, or None once it has gone."""
    if seen.get(key) != problem:
        log(f"{key}: {problem or 'back'}")
        seen[key] = problem


def require(condition, message):
    """Raise ValueError with message unless condition holds."""
    if not condition:
        raise ValueError(message)


def make_sensor(config, sensors, nvml):
    """Return a function of the tick's values so far and its time, which reads a temperature.

    The sensor is coretemp's or another hwmon device's, an NVIDIA GPU's, the average over a time
    of a sensor above it, or the highest of several sensors above it.
    """
    for name in (config.get("average"), *config.get("max", [])):
        require(name is None or name in sensors, f"sensor {config['name']}: no {name} above it")
    if "hwmon" in config:
        return lambda _values, _now: hwmon_temperature(config["hwmon"], config["label"])
    if "nvidia" in config:
        return lambda _values, _now: nvml.temperature(config["nvidia"])
    if "average" in config:
        require(config.get("seconds", 0) > 0, f"sensor {config['name']}: no seconds to average")
        samples = deque()

        def average(values, now):
            if values[config["average"]] is not None:
                samples.append((now, values[config["average"]]))
            while samples and samples[0][0] <= now - config["seconds"]:
                samples.popleft()
            if not samples:
                raise MissingError(f"no reading of {config['average']} in {config['seconds']} s")
            return sum(value for _, value in samples) / len(samples)

        return average
    require("max" in config, f"sensor {config['name']}: no hwmon, nvidia, average or max")

    def highest(values, _now):
        missing = [name for name in config["max"] if values[name] is None]
        if missing:
            raise MissingError(f"no reading of {', '.join(missing)}")
        return max(values[name] for name in config["max"])

    return highest


class Curve:
    """A graph of speed by temperature, which holds still within its hysteresis.

    The speed follows the temperature once it has moved the hysteresis away from where the speed
    last changed, and stayed past it for the response time; up and down each have their own. At or
    beyond the first or the last point, the speed follows at once.
    """

    def __init__(self, config, sensors):
        """Read the curve's config, checking it."""
        self.name, self.sensor, self.points = config["name"], config["sensor"], config["points"]
        self.hysteresis = {way: float(config["hysteresis"][way]) for way in WAYS}
        self.response = {way: float(config["response"][way]) for way in WAYS}
        require(self.sensor in sensors, f"curve {self.name}: no sensor {self.sensor}")
        temperatures = [temperature for temperature, _ in self.points]
        require(
            temperatures and all(a < b for a, b in itertools.pairwise(temperatures)),
            f"curve {self.name}: its points' temperatures do not rise",
        )
        require(
            all(0 <= speed <= PERCENT_FULL for _, speed in self.points),
            f"curve {self.name}: a speed outside 0-100%",
        )
        self.temperature = self.speed = self.pending = None

    def speed_at(self, temperature):
        """Return the speed at temperature: flat before the first point and past the last."""
        for (t0, s0), (t1, s1) in itertools.pairwise(self.points):
            if t0 <= temperature <= t1:
                return s0 + (s1 - s0) * (temperature - t0) / (t1 - t0)
        return self.points[0][1] if temperature < self.points[0][0] else self.points[-1][1]

    def update(self, temperature, now):
        """Move the speed to the temperature, None where it is missing, as the hysteresis allows."""
        if temperature is None:
            self.temperature = self.speed = self.pending = None
            return
        inside = self.points[0][0] < temperature < self.points[-1][0]
        if self.temperature is not None and inside:
            way = "up" if temperature > self.temperature else "down"
            if abs(temperature - self.temperature) < self.hysteresis[way]:
                self.pending = None
                return
            if self.pending is None or self.pending[0] != way:
                self.pending = (way, now)
            if now - self.pending[1] < self.response[way]:
                return
        self.temperature, self.speed, self.pending = temperature, self.speed_at(temperature), None


class Control:
    """A fan on a curve or at a fixed speed, which moves step % a tick at most, up or down."""

    def __init__(self, config, curves, step):
        """Read the control's config, checking it."""
        self.name, self.step = config["name"], step
        self.curve, self.percent = curves.get(config.get("curve")), config.get("percent")
        require(
            (self.curve is None) != (self.percent is None),
            f"{self.name}: a curve that exists, or a percent, and not both",
        )
        self.fan = HwmonFan(config["hwmon"], config["label"])
        self.speed, self.held = None, True

    def tick(self, seen):
        """Step the fan towards its speed, or hand it back where that is missing."""
        target = self.curve.speed if self.curve else self.percent
        try:
            if target is None:
                raise MissingError(f"curve {self.curve.name} has no temperature")
            if self.speed is None:
                self.speed = self.fan.speed()
            highest, lowest = self.speed + self.step["up"], self.speed - self.step["down"]
            self.speed = max(lowest, min(highest, target))
            # Before the fan is set, which may take it from the board and fail after.
            self.held = True
            self.fan.set(self.speed)
        except MissingError as err:
            report(seen, self.name, f"{err}; the fan goes back to the board's curve")
            self.release()
            return
        report(seen, self.name, None)

    def release(self):
        """Hand the fan back to the board, once."""
        self.speed = None
        if self.held:
            self.held = False
            try:
                self.fan.release()
            except MissingError as err:
                log(f"{self.name}: {err}")


def load(path):
    """Return the interval, the sensors, the curves and the controls of the config at path."""
    config = json.loads(Path(path).read_text())
    nvml, sensors = Nvml(), {}
    for sensor in config["sensors"]:
        sensors[sensor["name"]] = make_sensor(sensor, sensors, nvml)
    curves = {curve["name"]: Curve(curve, sensors) for curve in config["curves"]}
    step = {way: float(config["step"][way]) for way in WAYS}
    controls = [Control(control, curves, step) for control in config["controls"]]
    return float(config["interval"]), sensors, curves, controls


def run(interval, sensors, curves, controls):
    """Read the sensors, move the curves and step the fans every interval seconds, without end."""
    seen, deadline = {}, time.monotonic()
    while True:
        now, values = time.monotonic(), {}
        for name, sensor in sensors.items():
            try:
                values[name] = sensor(values, now)
                report(seen, name, None)
            except MissingError as err:
                values[name] = None
                report(seen, name, str(err))
        for curve in curves.values():
            curve.update(values[curve.sensor], now)
        for control in controls:
            control.tick(seen)
        deadline = max(deadline + interval, time.monotonic())
        time.sleep(max(0, deadline - time.monotonic()))


def main():
    """Check the config, or run on it until SIGTERM, then hand every fan back."""
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--check", action="store_true", help="check the config, set no fan")
    parser.add_argument("config", type=Path)
    args = parser.parse_args()
    try:
        interval, sensors, curves, controls = load(args.config)
    except (OSError, ValueError, KeyError, TypeError) as err:
        sys.exit(f"{args.config}: {err!r}")
    if args.check:
        return
    signal.signal(signal.SIGTERM, lambda *_: sys.exit(0))
    log(f"{len(controls)} fans, every {interval} s, from {args.config}")
    try:
        run(interval, sensors, curves, controls)
    finally:
        for control in controls:
            control.release()


if __name__ == "__main__":
    main()
