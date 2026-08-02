package cli

// boolValues is flag vocabulary rather than a server enum: the API filters
// booleans on 1/0, but a flag reads better as true/false, so commands
// accept these and map them on the way out (see boolFilter).
var boolValues = []string{"true", "false"}
