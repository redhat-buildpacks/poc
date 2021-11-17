package test

import (
	"bytes"
	report "github.com/containers/common/pkg/report"
	"github.com/stretchr/testify/assert"
	"testing"
	"text/tabwriter"
)

type machineReporter struct {
	Name     string
	Default  bool
	Created  string
	Running  bool
	LastUp   string
	VMType   string
	CPUs     uint64
	Memory   string
	DiskSize string
}

func TestTemplate_Parse(t *testing.T) {
	headers := report.Headers(machineReporter{}, map[string]string{
		"LastUp":   "LAST UP",
		"VmType":   "VM TYPE",
		"CPUs":     "CPUS",
		"Memory":   "MEMORY",
		"DiskSize": "DISK SIZE",
	})

	listFormat := "{{.Name}}\t{{.VMType}}\t{{.Created}}\t{{.LastUp}}\t{{.CPUs}}\t{{.Memory}}\t{{.DiskSize}}\n"
	row := report.NormalizeFormat(listFormat)
	format := report.EnforceRange(row)

	var buf bytes.Buffer
	tmpl, e := report.NewTemplate("TestTemplate").Parse(format)
	assert.NoError(t, e)

	// Check the headers content
	err := tmpl.Execute(&buf, headers)
	assert.NoError(t, err)
	assert.Equal(t, "NAME\tVM TYPE\tCREATED\tLAST UP\tCPUS\tMEMORY\tDISK SIZE\n", buf.String())
	buf.Reset()

	// Check the machineReport
	machineReporter:= genMachineReporter()
	err = tmpl.Execute(&buf, machineReporter)
	assert.NoError(t, err)
	assert.Equal(t, "macvm*\tqemu\t6 hours ago\t5 hours ago\t1\t2.147GB\t10.74GB\n", buf.String())
	buf.Reset()

	// Use now a Tabwriter
	w := tabwriter.NewWriter(&buf, 0, 5, 0, '\t', 0)
	err = tmpl.Execute(w, machineReporter)
	assert.NoError(t, err)
	w.Flush()
	res := buf.String()
	assert.Equal(t, "macvm*\tqemu\t6 hours ago\t5 hours ago\t1\t2.147GB\t10.74GB\n", res)
}

func genMachineReporter() ([]*machineReporter) {
	vms := make([]*machineReporter,0)
	vm := new(machineReporter)
	vm.Default = false
	vm.Running = false
	vm.Name = "macvm*"
	vm.LastUp = "5 hours ago"
	vm.Created = "6 hours ago"
	vm.VMType = "qemu"
	vm.CPUs = 1
	vm.Memory = "2.147GB"
	vm.DiskSize = "10.74GB"
	vms = append(vms, vm)
	return vms
}

