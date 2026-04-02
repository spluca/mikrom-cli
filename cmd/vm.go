package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spluca/mikrom-cli/internal/api"
)

var vmCmd = &cobra.Command{
	Use:   "vm",
	Short: "Manage virtual machines",
}

var vmListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all VMs",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		page, _ := cmd.Flags().GetInt("page")
		pageSize, _ := cmd.Flags().GetInt("page-size")

		resp, err := newClient().ListVMs(page, pageSize)
		if err != nil {
			return err
		}

		if len(resp.Items) == 0 {
			fmt.Println("No VMs found")
			return nil
		}

		fmt.Printf("%-36s  %-20s  %-12s  %-4s  %-8s  %s\n", "ID", "NAME", "STATUS", "CPU", "MEM(MB)", "IP")
		for _, vm := range resp.Items {
			fmt.Printf("%-36s  %-20s  %-12s  %-4d  %-8d  %s\n",
				vm.ID, vm.Name, vm.Status, vm.VCPUCount, vm.MemoryMB, vm.IPAddress)
		}
		fmt.Printf("\nTotal: %d\n", resp.Total)
		return nil
	},
}

var vmGetCmd = &cobra.Command{
	Use:   "get <vm-id>",
	Short: "Get VM details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()

		vm, err := newClient().GetVM(args[0])
		if err != nil {
			return err
		}

		printVM(vm)
		return nil
	},
}

var vmCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new VM",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		name, _ := cmd.Flags().GetString("name")
		desc, _ := cmd.Flags().GetString("description")
		vcpus, _ := cmd.Flags().GetInt("vcpus")
		memory, _ := cmd.Flags().GetInt("memory")

		vm, err := newClient().CreateVM(api.CreateVMRequest{
			Name:        name,
			Description: desc,
			VCPUCount:   vcpus,
			MemoryMB:    memory,
		})
		if err != nil {
			return err
		}

		fmt.Printf("VM created: %s\n", vm.ID)
		printVM(vm)
		return nil
	},
}

var vmDeleteCmd = &cobra.Command{
	Use:   "delete <vm-id>",
	Short: "Delete a VM",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()

		if err := newClient().DeleteVM(args[0]); err != nil {
			return err
		}

		fmt.Printf("VM %s queued for deletion\n", args[0])
		return nil
	},
}

var vmStartCmd = &cobra.Command{
	Use:   "start <vm-id>",
	Short: "Start a VM",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()

		if err := newClient().StartVM(args[0]); err != nil {
			return err
		}

		fmt.Printf("VM %s queued to start\n", args[0])
		return nil
	},
}

var vmStopCmd = &cobra.Command{
	Use:   "stop <vm-id>",
	Short: "Stop a VM",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()

		if err := newClient().StopVM(args[0]); err != nil {
			return err
		}

		fmt.Printf("VM %s queued to stop\n", args[0])
		return nil
	},
}

var vmRestartCmd = &cobra.Command{
	Use:   "restart <vm-id>",
	Short: "Restart a VM",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()

		if err := newClient().RestartVM(args[0]); err != nil {
			return err
		}

		fmt.Printf("VM %s queued to restart\n", args[0])
		return nil
	},
}

var vmDeployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Build an app from source and deploy it as a VM",
	Long: `Clone a Git repository, build it with Cloud Native Buildpacks,
and provision a Firecracker microVM from the resulting image.

The VM starts in "building" status and transitions through
provisioning → running asynchronously. Poll with "mikrom vm get <id>".`,
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		name, _ := cmd.Flags().GetString("name")
		desc, _ := cmd.Flags().GetString("description")
		vcpus, _ := cmd.Flags().GetInt("vcpus")
		memory, _ := cmd.Flags().GetInt("memory")
		repoURL, _ := cmd.Flags().GetString("repo")
		builder, _ := cmd.Flags().GetString("builder")
		kernelPath, _ := cmd.Flags().GetString("kernel-path")

		vm, err := newClient().DeployVM(api.DeployVMRequest{
			Name:        name,
			Description: desc,
			VCPUCount:   vcpus,
			MemoryMB:    memory,
			RepoURL:     repoURL,
			Builder:     builder,
			KernelPath:  kernelPath,
		})
		if err != nil {
			return err
		}

		fmt.Printf("Deploy queued: %s (status: %s)\n", vm.ID, vm.Status)
		fmt.Printf("Poll status with: mikrom vm get %s\n", vm.ID)
		printVM(vm)
		return nil
	},
}

func printVM(vm *api.VM) {
	fmt.Printf("ID:          %s\n", vm.ID)
	fmt.Printf("Name:        %s\n", vm.Name)
	fmt.Printf("Description: %s\n", vm.Description)
	fmt.Printf("Status:      %s\n", vm.Status)
	fmt.Printf("vCPUs:       %d\n", vm.VCPUCount)
	fmt.Printf("Memory:      %d MB\n", vm.MemoryMB)
	fmt.Printf("IP Address:  %s\n", vm.IPAddress)
}

func init() {
	vmListCmd.Flags().Int("page", 1, "Page number")
	vmListCmd.Flags().Int("page-size", 20, "Items per page")

	vmCreateCmd.Flags().String("name", "", "VM name")
	vmCreateCmd.Flags().String("description", "", "VM description")
	vmCreateCmd.Flags().Int("vcpus", 1, "Number of vCPUs (1-32)")
	vmCreateCmd.Flags().Int("memory", 512, "Memory in MB (128-32768)")
	vmCreateCmd.MarkFlagRequired("name")

	vmDeployCmd.Flags().String("name", "", "VM name")
	vmDeployCmd.Flags().String("description", "", "VM description")
	vmDeployCmd.Flags().Int("vcpus", 2, "Number of vCPUs (1-32)")
	vmDeployCmd.Flags().Int("memory", 1024, "Memory in MB (128-32768)")
	vmDeployCmd.Flags().String("repo", "", "Public Git repository URL to build and deploy")
	vmDeployCmd.Flags().String("builder", "", "Buildpack builder image (default: paketobuildpacks/builder:base)")
	vmDeployCmd.Flags().String("kernel-path", "", "Path to the kernel on the firecracker-agent host (optional)")
	vmDeployCmd.MarkFlagRequired("name")
	vmDeployCmd.MarkFlagRequired("repo")

	vmCmd.AddCommand(vmListCmd)
	vmCmd.AddCommand(vmGetCmd)
	vmCmd.AddCommand(vmCreateCmd)
	vmCmd.AddCommand(vmDeployCmd)
	vmCmd.AddCommand(vmDeleteCmd)
	vmCmd.AddCommand(vmStartCmd)
	vmCmd.AddCommand(vmStopCmd)
	vmCmd.AddCommand(vmRestartCmd)
}
