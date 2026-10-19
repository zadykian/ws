# Written by github.com/zadykian/ws, whose next run replaces any change made here.
"""ws-fans' devices: hwmon's temperatures and fans, and NVIDIA GPUs' temperatures.

A device that cannot be read or set raises MissingError, and ws-fans tries it again on its next
tick: a driver may load after the service starts, or come back after an update.
"""

import ctypes
from pathlib import Path

HWMON = Path("/sys/class/hwmon")
# The values of pwmN_enable that hand the fan to pwmN, and back to the board's own curve.
PWM_MANUAL = 1
PWM_AUTO = 2
PWM_FULL = 255
PERCENT_FULL = 100
NVML_SUCCESS = 0
NVML_TEMPERATURE_GPU = 0


class MissingError(Exception):
    """A device that cannot be read or set now."""


def read(path):
    """Return the stripped text of the sysfs file at path."""
    try:
        return path.read_text().strip()
    except OSError as err:
        raise MissingError(f"cannot read {path}: {err.strerror}") from err


def write(path, value):
    """Write value to the sysfs file at path."""
    try:
        path.write_text(f"{value}\n")
    except OSError as err:
        raise MissingError(f"cannot write {value} to {path}: {err.strerror}") from err


def hwmon_file(device, kind, label, attribute):
    """Return a file of the channel of kind, temp or fan, labelled label on the hwmon device.

    attribute is the file's name with {} for the channel's number, as pwm{}. The channels are
    looked up each time, as hwmon numbers its devices anew when a driver loads again.
    """
    for path in sorted(HWMON.glob("hwmon*")):
        if read(path / "name") != device:
            continue
        for label_file in path.glob(f"{kind}*_label"):
            if read(label_file) == label:
                number = label_file.name.removeprefix(kind).removesuffix("_label")
                return path / attribute.format(number)
    raise MissingError(f"no {kind} labelled {label} on a hwmon device {device}")


def hwmon_temperature(device, label):
    """Return the temperature in °C of the channel labelled label on the hwmon device."""
    return int(read(hwmon_file(device, "temp", label, "temp{}_input"))) / 1000


class Nvml:
    """NVIDIA's management library, loaded at its first use, or later where the driver is not up."""

    def __init__(self):
        """Start with the library not loaded."""
        self.lib = None
        self.handles = {}

    def call(self, function, *args):
        """Call NVML's function with args."""
        if self.lib is None:
            try:
                lib = ctypes.CDLL("libnvidia-ml.so.1")
            except OSError as err:
                raise MissingError(f"no NVML: {err}") from err
            if (code := lib.nvmlInit_v2()) != NVML_SUCCESS:
                raise MissingError(f"NVML's nvmlInit_v2 failed with error {code}")
            self.lib = lib
        if (code := getattr(self.lib, function)(*args)) != NVML_SUCCESS:
            raise MissingError(f"NVML's {function} failed with error {code}")

    def handle(self, gpu):
        """Return the handle of the GPU of index gpu."""
        if gpu not in self.handles:
            handle = ctypes.c_void_p()
            self.call("nvmlDeviceGetHandleByIndex_v2", ctypes.c_uint(gpu), ctypes.byref(handle))
            self.handles[gpu] = handle
        return self.handles[gpu]

    def temperature(self, gpu):
        """Return the temperature in °C of the GPU gpu."""
        value, kind = ctypes.c_uint(), ctypes.c_uint(NVML_TEMPERATURE_GPU)
        self.call("nvmlDeviceGetTemperature", self.handle(gpu), kind, ctypes.byref(value))
        return value.value


class HwmonFan:
    """A fan header of a hwmon device, such as nct6687's, found by its fan's label."""

    def __init__(self, device, label):
        """Name the fan, which is not touched yet."""
        self.device, self.label, self.written = device, label, None

    def file(self, attribute):
        """Return the fan's file of attribute, as pwm{}."""
        return hwmon_file(self.device, "fan", self.label, attribute)

    def speed(self):
        """Return the speed the fan is set to now, in %."""
        return int(read(self.file("pwm{}"))) * PERCENT_FULL / PWM_FULL

    def set(self, speed):
        """Set the fan to speed %, unless it is set to it already."""
        value = round(speed * PWM_FULL / PERCENT_FULL)
        if self.written is None:
            write(self.file("pwm{}_enable"), PWM_MANUAL)
        if value != self.written:
            write(self.file("pwm{}"), value)
            self.written = value

    def release(self):
        """Hand the fan back to the board's curve."""
        self.written = None
        write(self.file("pwm{}_enable"), PWM_AUTO)
