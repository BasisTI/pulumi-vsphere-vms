package main

import (
	"github.com/BasisTI/pulumi-vsphere-vms/vsphere-vms"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Create the VMs
		vsphereVms, err := vsphere_vms.NewVsphereVmsFromConfig(ctx, "test-vms")
		if err != nil {
			return err
		}

		// Export VM IDs
		var vmIds []pulumi.StringOutput
		for _, vm := range vsphereVms.VirtualMachines {
			vmIds = append(vmIds, vm.ID().ToStringOutput())
		}
		ctx.Export("vmIds", pulumi.ToStringArrayOutput(vmIds))
		return nil
	})
}
