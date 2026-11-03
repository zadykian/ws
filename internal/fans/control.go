package fans

// control is a fan on a curve or at a fixed speed, which moves Step % a tick at most.
type control struct {
	name    string
	fan     Fan
	curve   *Curve
	percent *float64
	step    Pair
	speed   float64
	has     bool // speed is where the fan is
	held    bool // the fan may be off the board's curve; a daemon starts with every fan held, as a killed one may have left it
}

// tick steps the fan towards its speed, or hands it back where a device or a temperature is
// missing. It returns the missing problem, or "".
func (c *control) tick() (problem string, err error) {
	var target float64
	switch {
	case c.curve != nil:
		t, ok := c.curve.Speed()
		if !ok {
			return c.giveBack(missing("curve %s has no temperature", c.curve.name))
		}
		target = t
	default:
		target = *c.percent
	}
	if !c.has {
		s, err := c.fan.Speed()
		if err != nil {
			return c.giveBack(err)
		}
		c.speed, c.has = s, true
	}
	c.speed = max(c.speed-c.step.Down, min(c.speed+c.step.Up, target))
	// Before the fan is set, which may take it from the board and fail after.
	c.held = true
	if err := c.fan.Set(c.speed); err != nil {
		return c.giveBack(err)
	}
	return "", nil
}

// giveBack hands the fan back and returns err's text, where err is a missing error. Any other
// error is returned.
func (c *control) giveBack(err error) (string, error) {
	if !IsMissing(err) {
		return "", err
	}
	if rerr := c.release(); rerr != nil {
		return "", rerr
	}
	return err.Error() + "; the fan goes back to the board's curve", nil
}

// release hands the fan back to the board, once. A missing fan is not an error.
func (c *control) release() error {
	c.has = false
	if !c.held {
		return nil
	}
	c.held = false
	if err := c.fan.Release(); err != nil && !IsMissing(err) {
		return err
	}
	return nil
}
