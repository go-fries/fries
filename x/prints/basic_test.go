package prints

import "testing"

func TestInfo(*testing.T) {
	Line("Line: Hello World!", "Hello Go!")                    //nolint:errcheck
	Linef("Linef: %s %s\n", "Hello World!", "Hello Go!")       //nolint:errcheck
	NewLine()                                                  //nolint:errcheck
	Info("Info: Hello World!", "Hello Go!")                    //nolint:errcheck
	Infof("Infof: %s %s\n", "Hello World!", "Hello Go!")       //nolint:errcheck
	Comment("Comment: Hello World!", "Hello Go!")              //nolint:errcheck
	Commentf("Commentf: %s %s\n", "Hello World!", "Hello Go!") //nolint:errcheck
	Error("Error: Hello World!", "Hello Go!")                  //nolint:errcheck
	Errorf("Errorf: %s %s\n", "Hello World!", "Hello Go!")     //nolint:errcheck
	Warn("Warn: Hello World!", "Hello Go!")                    //nolint:errcheck
	Warnf("Warnf: %s %s\n", "Hello World!", "Hello Go!")       //nolint:errcheck
	Alert("Alert: Hello World!", "Hello Go!")                  //nolint:errcheck
	Alertf("Alertf: %s %s\n", "Hello World!", "Hello Go!")     //nolint:errcheck
}
