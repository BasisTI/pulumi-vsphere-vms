# Pulumi vSphere VMs Example

This example demonstrates how to use the `pulumi-vsphere-vms` component to create virtual machines in a vSphere environment.

## Prerequisites

1. vSphere environment with:
   - vCenter Server accessible
   - Ubuntu 24.04 template VM
   - Target datacenter, cluster, and datastore
   - Network configured

2. Pulumi CLI installed
3. Go 1.24+ installed

## Configuration

Update the values in `main.go` to match your environment:

- **vSphere Configuration**: Datacenter, cluster, datastore, network names
- **Network Configuration**: Gateway, DNS servers, domain, subnet mask
- **VM Configuration**: Names, hostnames, IP addresses, CPU, memory

## Running the Example

1. Initialize the Pulumi project:
   ```bash
   cd example
   pulumi stack init dev
   ```

2. Configure vSphere credentials:
   ```bash
   pulumi config set vsphere:host "vcenter.example.com"
   pulumi config set vsphere:user "administrator@vsphere.local"
   pulumi config set vsphere:password --secret "your-password"
   pulumi config set vsphere:allowUnverifiedSsl true  # if needed
   ```

3. Run the example:
   ```bash
   pulumi up
   ```

## Network Configuration Fix

This example includes the recent fix for network configuration issues:

- ✅ DNS servers are now properly configured at the network interface level
- ✅ Reasonable timeouts (300 seconds) prevent infinite hangs
- ✅ Complete netplan configuration for Ubuntu 24.04

## Testing the Fix

The component now properly configures:
- Network interface with IP address and netmask
- DNS servers at the interface level
- Gateway at the global level
- Domain configuration

This should resolve the issue where network interfaces stayed down due to incomplete netplan configuration.

## Cleanup

```bash
pulumi destroy
```