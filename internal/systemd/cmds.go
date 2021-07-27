// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package systemd

// LogCmds represents the command and base arguments to retrieve a unit's logs
func LogCmds(unit string) []string {
	// -m --merge	     Show entries from all available journals
	// -u --unit=UNIT    Show logs from the specified unit
	return []string{"journalctl", "-m", "-u", unit}
}

// RestartCmds represents the command and base arguments to restart a unit
func RestartCmds(unit string) []string {
	return []string{"systemctl", "restart", unit}
}

// StartCmds represents the command and base arguments to start a unit
func StartCmds(unit string) []string {
	return []string{"systemctl", "start", unit}
}
