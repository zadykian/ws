package values

func defaultFans() Fans {
	zero := 0
	pair := FanPair{Up: 2, Down: 2}
	resp := FanPair{Up: 5, Down: 5}
	return Fans{
		BoardVendor: "Micro-Star International Co., Ltd.",
		Sensors: []FanSensor{
			{Name: "cpu", Hwmon: "coretemp", Label: "Package id 0"},
			{Name: "gpu", Nvidia: &zero},
			{Name: "cpu_average", Average: "cpu", Seconds: 60},
			{Name: "gpu_average", Average: "gpu", Seconds: 60},
			{Name: "cpu_gpu_max", Max: []string{"cpu_average", "gpu_average"}},
		},
		Curves: []FanCurve{
			{Name: "cpu", Sensor: "cpu_average", Points: [][2]int{{50, 30}, {80, 60}}, Hysteresis: pair, Response: resp},
			{Name: "gpu", Sensor: "gpu_average", Points: [][2]int{{30, 20}, {50, 20}, {80, 60}}, Hysteresis: pair, Response: resp},
			{Name: "cpu_gpu_max", Sensor: "cpu_gpu_max", Points: [][2]int{{50, 30}, {80, 60}}, Hysteresis: pair, Response: resp},
		},
		Controls: []FanControl{
			{Name: "CPU Pump", Hwmon: "nct6687", Label: "Pump Fan", Curve: "cpu"},
			{Name: "Exhaust Fans", Hwmon: "nct6687", Label: "CPU Fan", Curve: "cpu_gpu_max"},
			{Name: "GPU Fans", Hwmon: "nct6687", Label: "System Fan #1", Curve: "gpu"},
			{Name: "MB Fan", Hwmon: "nct6687", Label: "System Fan #2", Percent: &zero},
		},
		Step: FanPair{Up: 10, Down: 10}, Interval: 1,
	}
}
