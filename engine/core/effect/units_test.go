package effect

import "testing"

func TestCheckUnitName(t *testing.T) {
	for name, ok := range map[string]bool{
		"docker.service": true, "apt-daily.timer": true, "docker.socket": true, "multi-user.target": true,
		"docker": false, "": false, ".service": false, "docker.conf": false, "ssh.service.d": false,
	} {
		err := CheckUnitName("start", name)
		if (err == nil) != ok {
			t.Errorf("CheckUnitName(%q) = %v, want ok %v", name, err, ok)
		}
	}
	if err := CheckUnitName("start", "docker"); err == nil || err.Error() == "" ||
		err.Error()[:5] != "start" {
		t.Errorf("the error does not name the call: %v", err)
	}
}
