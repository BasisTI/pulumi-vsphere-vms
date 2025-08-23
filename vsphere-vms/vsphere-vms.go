package vsphere_vms

import (
	"github.com/pulumi/pulumi-vsphere/sdk/v4/go/vsphere"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// VmData defines the structure for a virtual machine's data.
// It includes specifications for the VM's name, network configuration, and hardware resources.
// Each field is tagged with `yaml` annotations for easy deserialization from configuration files.
type VmData struct {
	Name        string `yaml:"name"`        // Name of the virtual machine.
	HostName    string `yaml:"hostName"`    // Hostname of the virtual machine.
	Ipv4Address string `yaml:"ipv4address"` // IPv4 address of the virtual machine.
	NumCpus     int    `yaml:"numCpus"`     // Number of CPUs for the virtual machine.
	Memory      int    `yaml:"memory"`      // Memory size in MB for the virtual machine.
}

// VsphereCfg defines the vSphere-specific configuration required for creating virtual machines.
// This includes details about the vSphere environment, such as datacenter, datastore, and cluster information.
// It also specifies the template to be used for cloning new VMs.
type VsphereCfg struct {
	Datacenter     string `yaml:"datacenter"`     // Name of the vSphere datacenter.
	Datastore      string `yaml:"datastore"`      // Name of the vSphere datastore.
	Cluster        string `yaml:"cluster"`        // Name of the vSphere cluster.
	NetworkName    string `yaml:"networkName"`    // Name of the vSphere network.
	TemplateName   string `yaml:"templateName"`   // Name of the VM template to clone from.
	TemplateFolder string `yaml:"templateFolder"` // Folder containing the VM template.
	VmsFolder      string `yaml:"vmsFolder"`      // Folder to place the new virtual machines in.
	EnableLogging  bool   `yaml:"enableLogging"`  // Enable logging for the virtual machine.
	EnableDiskUuid bool   `yaml:"enableDiskUuid"` // Enable disk UUID for the virtual machine.
}

// NetworkCfg defines the network configuration for the virtual machines.
// This includes gateway, DNS servers, DNS suffixes, domain, and subnet mask.
type NetworkCfg struct {
	Gateway     string   `yaml:"gateway"`     // Network gateway IP address.
	DnsServers  []string `yaml:"dnsServers"`  // List of DNS server IP addresses.
	DnsSuffixes []string `yaml:"dnsSuffixes"` // List of DNS suffixes.
	Domain      string   `yaml:"domain"`      // Network domain name.
	Mask        int      `yaml:"mask"`        // Subnet mask.
}

// LookupData holds the results of vSphere lookups.
// It stores references to the datacenter, cluster, datastore, template VM, and network.
// This data is used to provision new virtual machines.
type LookupData struct {
	Datacenter *vsphere.LookupDatacenterResult
	Cluster    *vsphere.LookupComputeClusterResult
	Datastore  *vsphere.GetDatastoreResult
	TemplateVm *vsphere.LookupVirtualMachineResult
	Network    *vsphere.GetNetworkResult
}

// VsphereVms is a Pulumi component resource for managing a group of vSphere virtual machines.
// It encapsulates the logic for creating multiple VMs based on a shared configuration.
type VsphereVms struct {
	pulumi.ResourceState
	VirtualMachines []*vsphere.VirtualMachine // List of created virtual machines.
}

// VsphereVmsArgs defines the arguments for creating a VsphereVms component.
// It includes a list of VM data, vSphere configuration, and network configuration.
type VsphereVmsArgs struct {
	Vms        []VmData
	VsphereCfg VsphereCfg
	NetworkCfg NetworkCfg
}

// NewVsphereVmsFromConfig creates a new VsphereVms component by reading configuration from Pulumi config.
// It automatically reads the "vms", "vsphereCfg", and "networkCfg" configuration objects.
func NewVsphereVmsFromConfig(ctx *pulumi.Context, name string, opts ...pulumi.ResourceOption) (*VsphereVms, error) {
	var vsphereVmsArgs VsphereVmsArgs
	cfg := config.New(ctx, "")
	cfg.RequireObject("vms", &vsphereVmsArgs.Vms)
	cfg.RequireObject("vsphereCfg", &vsphereVmsArgs.VsphereCfg)
	cfg.RequireObject("networkCfg", &vsphereVmsArgs.NetworkCfg)

	return NewVsphereVms(ctx, name, &vsphereVmsArgs, opts...)
}

// NewVsphereVms creates a new VsphereVms component.
// It registers the component with Pulumi and creates the virtual machines based on the provided arguments.
func NewVsphereVms(ctx *pulumi.Context, name string, args *VsphereVmsArgs, opts ...pulumi.ResourceOption) (*VsphereVms, error) {
	vsphereVms := &VsphereVms{}
	err := ctx.RegisterComponentResource("pkg:index:VsphereVms", name, vsphereVms, opts...)
	if err != nil {
		return nil, err
	}

	lookupData, err := lookupData(ctx, args.VsphereCfg, args.NetworkCfg)
	if err != nil {
		return nil, err
	}

	var virtualMachines []*vsphere.VirtualMachine
	for _, vm := range args.Vms {
		newVm, err := createVm(ctx, lookupData, args.NetworkCfg, args.VsphereCfg, vm, pulumi.Parent(vsphereVms))
		if err != nil {
			return nil, err
		}
		virtualMachines = append(virtualMachines, newVm)
	}

	vsphereVms.VirtualMachines = virtualMachines

	if err := ctx.RegisterResourceOutputs(vsphereVms, pulumi.Map{}); err != nil {
		return nil, err
	}

	return vsphereVms, nil
}

// createVm creates a single virtual machine in vSphere.
// It uses the provided lookup data and configuration to clone a new VM from a template.
func createVm(ctx *pulumi.Context, lookupData *LookupData, networkCfg NetworkCfg, vsphereCfg VsphereCfg, vm VmData, opts ...pulumi.ResourceOption) (*vsphere.VirtualMachine, error) {
	templateVm := lookupData.TemplateVm
	const net_timeout = 300
	newVm, err := vsphere.NewVirtualMachine(ctx, vm.Name, &vsphere.VirtualMachineArgs{
		Name:                   pulumi.String(vm.Name),
		ResourcePoolId:         pulumi.String(lookupData.Cluster.ResourcePoolId),
		DatastoreId:            pulumi.String(lookupData.Datastore.Id),
		NumCpus:                pulumi.Int(vm.NumCpus),
		Memory:                 pulumi.Int(vm.Memory),
		Clone:                  getVMCloneArgs(lookupData, networkCfg, vm),
		Disks:                  getVmCloneDiskArray(templateVm),
		NetworkInterfaces:      getVmCloneNetworkInterfaceArray(templateVm, lookupData.Network),
		EfiSecureBootEnabled:   pulumi.BoolPtrFromPtr(templateVm.EfiSecureBootEnabled),
		EnableLogging:          pulumi.Bool(vsphereCfg.EnableLogging),
		EnableDiskUuid:         pulumi.Bool(vsphereCfg.EnableDiskUuid),
		Firmware:               pulumi.StringPtrFromPtr(templateVm.Firmware),
		Folder:                 pulumi.String(vsphereCfg.VmsFolder),
		GuestId:                pulumi.String("ubuntu64Guest"),
		WaitForGuestIpTimeout:  pulumi.Int(net_timeout),
		WaitForGuestNetTimeout: pulumi.Int(net_timeout),
	}, opts...)
	if err != nil {
		return nil, err
	}
	return newVm, nil
}

// getVmCloneNetworkInterfaceArray creates a network interface array for the new VM.
func getVmCloneNetworkInterfaceArray(templateVm *vsphere.LookupVirtualMachineResult, network *vsphere.GetNetworkResult) vsphere.VirtualMachineNetworkInterfaceArray {
	return vsphere.VirtualMachineNetworkInterfaceArray{
		&vsphere.VirtualMachineNetworkInterfaceArgs{
			AdapterType: pulumi.String(templateVm.NetworkInterfaces[0].AdapterType),
			NetworkId:   pulumi.String(network.Id),
		},
	}
}

// getVmCloneDiskArray creates a disk array for the new VM.
func getVmCloneDiskArray(templateVm *vsphere.LookupVirtualMachineResult) vsphere.VirtualMachineDiskArray {
	return vsphere.VirtualMachineDiskArray{
		&vsphere.VirtualMachineDiskArgs{
			Label:           pulumi.String("disk0"),
			EagerlyScrub:    pulumi.Bool(templateVm.Disks[0].EagerlyScrub),
			Size:            pulumi.Int(templateVm.Disks[0].Size),
			ThinProvisioned: pulumi.Bool(templateVm.Disks[0].ThinProvisioned),
		},
	}
}

// getVMCloneArgs creates the clone arguments for the new VM.
func getVMCloneArgs(lookupData *LookupData, networkCfg NetworkCfg, vm VmData) *vsphere.VirtualMachineCloneArgs {
	return &vsphere.VirtualMachineCloneArgs{
		TemplateUuid: pulumi.String(lookupData.TemplateVm.Id),
		Customize: &vsphere.VirtualMachineCloneCustomizeArgs{
			DnsServerLists: toStringArray(networkCfg.DnsServers),
			DnsSuffixLists: toStringArray(networkCfg.DnsSuffixes),
			Ipv4Gateway:    pulumi.String(networkCfg.Gateway),
			LinuxOptions: &vsphere.VirtualMachineCloneCustomizeLinuxOptionsArgs{
				Domain:   pulumi.String(networkCfg.Domain),
				HostName: pulumi.String(vm.HostName),
			},
			NetworkInterfaces: vsphere.VirtualMachineCloneCustomizeNetworkInterfaceArray{
				&vsphere.VirtualMachineCloneCustomizeNetworkInterfaceArgs{
					DnsDomain:      pulumi.String(networkCfg.Domain),
					DnsServerLists: toStringArray(networkCfg.DnsServers),
					Ipv4Address:    pulumi.String(vm.Ipv4Address),
					Ipv4Netmask:    pulumi.Int(networkCfg.Mask),
				},
			},
		},
	}
}

// toStringArray converts a string slice to a pulumi.StringArray.
func toStringArray(list []string) pulumi.StringArray {
	dnsSuffixes := pulumi.StringArray{}
	for _, item := range list {
		dnsSuffixes = append(dnsSuffixes, pulumi.String(item))
	}
	return dnsSuffixes
}

// lookupData performs the necessary vSphere lookups to get the required resources for VM creation.
func lookupData(ctx *pulumi.Context, vsphereCfg VsphereCfg, networkCfg NetworkCfg) (*LookupData, error) {
	datacenter, err := vsphere.LookupDatacenter(ctx, &vsphere.LookupDatacenterArgs{
		Name: pulumi.StringRef(vsphereCfg.Datacenter),
	}, nil)
	if err != nil {
		return nil, err
	}
	cluster, err := vsphere.LookupComputeCluster(ctx, &vsphere.LookupComputeClusterArgs{
		DatacenterId: pulumi.StringRef(datacenter.Id),
		Name:         vsphereCfg.Cluster,
	}, nil)
	if err != nil {
		return nil, err
	}
	datastore, err := vsphere.GetDatastore(ctx, &vsphere.GetDatastoreArgs{
		DatacenterId: pulumi.StringRef(datacenter.Id),
		Name:         vsphereCfg.Datastore,
	})
	if err != nil {
		return nil, err
	}
	templateVm, err := vsphere.LookupVirtualMachine(ctx, &vsphere.LookupVirtualMachineArgs{
		DatacenterId: pulumi.StringRef(datacenter.Id),
		Name:         pulumi.StringRef(vsphereCfg.TemplateName),
		Folder:       pulumi.StringRef(vsphereCfg.TemplateFolder),
	})
	if err != nil {
		return nil, err
	}
	network, err := vsphere.GetNetwork(ctx, &vsphere.GetNetworkArgs{
		DatacenterId: pulumi.StringRef(datacenter.Id),
		Name:         vsphereCfg.NetworkName,
	})
	if err != nil {
		return nil, err
	}
	return &LookupData{
		Datacenter: datacenter,
		Cluster:    cluster,
		Datastore:  datastore,
		TemplateVm: templateVm,
		Network:    network,
	}, nil
}
