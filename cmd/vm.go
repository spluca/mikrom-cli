package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

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
		status, _ := cmd.Flags().GetString("status")

		resp, err := newClient().ListVMs(page, pageSize, status)
		if err != nil {
			return err
		}

		if isJSON() {
			data, _ := json.MarshalIndent(resp, "", "  ")
			fmt.Println(string(data))
			return nil
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

		if isJSON() {
			data, _ := json.MarshalIndent(vm, "", "  ")
			fmt.Println(string(data))
			return nil
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
		kernelPath, _ := cmd.Flags().GetString("kernel-path")
		rootfsPath, _ := cmd.Flags().GetString("rootfs-path")
		imageRef, _ := cmd.Flags().GetString("image-ref")
		wait, _ := cmd.Flags().GetBool("wait")

		vm, err := newClient().CreateVM(api.CreateVMRequest{
			Name:        name,
			Description: desc,
			VCPUCount:   vcpus,
			MemoryMB:    memory,
			KernelPath:  kernelPath,
			RootfsPath:  rootfsPath,
			ImageRef:    imageRef,
		})
		if err != nil {
			return err
		}

		if wait {
			fmt.Fprintf(os.Stderr, "Waiting for VM %s to reach running...", vm.ID)
			vm, err = waitForVM(newClient(), vm.ID, "running", 5*time.Minute)
			if err != nil {
				return err
			}
		}

		if isJSON() {
			data, _ := json.MarshalIndent(vm, "", "  ")
			fmt.Println(string(data))
			return nil
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
		wait, _ := cmd.Flags().GetBool("wait")

		if err := newClient().DeleteVM(args[0]); err != nil {
			return err
		}

		if wait {
			fmt.Fprintf(os.Stderr, "Waiting for VM %s to be deleted...", args[0])
			if err := waitForVMDeleted(newClient(), args[0], 5*time.Minute); err != nil {
				return err
			}
			fmt.Println()
			fmt.Printf("VM %s deleted\n", args[0])
			return nil
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
		wait, _ := cmd.Flags().GetBool("wait")

		if err := newClient().StartVM(args[0]); err != nil {
			return err
		}

		if wait {
			fmt.Fprintf(os.Stderr, "Waiting for VM %s to start...", args[0])
			vm, err := waitForVM(newClient(), args[0], "running", 5*time.Minute)
			if err != nil {
				return err
			}
			if isJSON() {
				data, _ := json.MarshalIndent(vm, "", "  ")
				fmt.Println(string(data))
				return nil
			}
			printVM(vm)
			return nil
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
		wait, _ := cmd.Flags().GetBool("wait")

		if err := newClient().StopVM(args[0]); err != nil {
			return err
		}

		if wait {
			fmt.Fprintf(os.Stderr, "Waiting for VM %s to stop...", args[0])
			vm, err := waitForVM(newClient(), args[0], "stopped", 5*time.Minute)
			if err != nil {
				return err
			}
			if isJSON() {
				data, _ := json.MarshalIndent(vm, "", "  ")
				fmt.Println(string(data))
				return nil
			}
			printVM(vm)
			return nil
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
		wait, _ := cmd.Flags().GetBool("wait")

		if err := newClient().RestartVM(args[0]); err != nil {
			return err
		}

		if wait {
			fmt.Fprintf(os.Stderr, "Waiting for VM %s to restart...", args[0])
			vm, err := waitForVM(newClient(), args[0], "running", 5*time.Minute)
			if err != nil {
				return err
			}
			if isJSON() {
				data, _ := json.MarshalIndent(vm, "", "  ")
				fmt.Println(string(data))
				return nil
			}
			printVM(vm)
			return nil
		}

		fmt.Printf("VM %s queued to restart\n", args[0])
		return nil
	},
}

var vmUpdateCmd = &cobra.Command{
	Use:   "update <vm-id>",
	Short: "Update VM metadata",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()

		req := api.UpdateVMRequest{}
		if cmd.Flags().Changed("name") {
			name, _ := cmd.Flags().GetString("name")
			req.Name = &name
		}
		if cmd.Flags().Changed("description") {
			desc, _ := cmd.Flags().GetString("description")
			req.Description = &desc
		}

		vm, err := newClient().UpdateVM(args[0], req)
		if err != nil {
			return err
		}

		if isJSON() {
			data, _ := json.MarshalIndent(vm, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		printVM(vm)
		return nil
	},
}

var vmDeployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Build an app from source and deploy it as a VM",
	Long: `Clone a Git repository, build it with Cloud Native Buildpacks,
and provision a Firecracker microVM from the resulting image.

The VM starts in "building" status and transitions through
provisioning → running asynchronously. Use --wait to block until running.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		name, _ := cmd.Flags().GetString("name")
		desc, _ := cmd.Flags().GetString("description")
		vcpus, _ := cmd.Flags().GetInt("vcpus")
		memory, _ := cmd.Flags().GetInt("memory")
		repoURL, _ := cmd.Flags().GetString("repo")
		builder, _ := cmd.Flags().GetString("builder")
		kernelPath, _ := cmd.Flags().GetString("kernel-path")
		wait, _ := cmd.Flags().GetBool("wait")

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

		if wait {
			fmt.Fprintf(os.Stderr, "Waiting for VM %s to reach running...", vm.ID)
			vm, err = waitForVM(newClient(), vm.ID, "running", 15*time.Minute)
			if err != nil {
				return err
			}
		}

		if isJSON() {
			data, _ := json.MarshalIndent(vm, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Deploy queued: %s (status: %s)\n", vm.ID, vm.Status)
		if !wait {
			fmt.Printf("Poll status with: mikrom vm get %s\n", vm.ID)
		}
		printVM(vm)
		return nil
	},
}

// waitForVM polls GetVM every 2 seconds until the VM reaches targetStatus or
// "error", or until the timeout elapses. Progress dots are written to stderr.
func waitForVM(client *api.Client, vmID, targetStatus string, timeout time.Duration) (*api.VM, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		time.Sleep(2 * time.Second)
		vm, err := client.GetVM(vmID)
		if err != nil {
			return nil, err
		}
		fmt.Fprint(os.Stderr, ".")
		if vm.Status == targetStatus {
			fmt.Fprintln(os.Stderr, " done")
			return vm, nil
		}
		if vm.Status == "error" {
			fmt.Fprintln(os.Stderr, " error")
			return nil, fmt.Errorf("VM %s entered error state", vmID)
		}
	}
	return nil, fmt.Errorf("timed out waiting for VM %s to reach %q", vmID, targetStatus)
}

// waitForVMDeleted polls GetVM until it returns an error (404), indicating
// the VM has been fully removed.
func waitForVMDeleted(client *api.Client, vmID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		time.Sleep(2 * time.Second)
		_, err := client.GetVM(vmID)
		fmt.Fprint(os.Stderr, ".")
		if err != nil {
			// Any error here likely means 404 = deleted.
			fmt.Fprintln(os.Stderr, " done")
			return nil
		}
	}
	return fmt.Errorf("timed out waiting for VM %s to be deleted", vmID)
}

func printVM(vm *api.VM) {
	fmt.Printf("ID:          %s\n", vm.ID)
	fmt.Printf("Name:        %s\n", vm.Name)
	fmt.Printf("Description: %s\n", vm.Description)
	fmt.Printf("Status:      %s\n", vm.Status)
	fmt.Printf("vCPUs:       %d\n", vm.VCPUCount)
	fmt.Printf("Memory:      %d MB\n", vm.MemoryMB)
	fmt.Printf("IP Address:  %s\n", vm.IPAddress)
	if vm.ErrorMessage != "" {
		fmt.Printf("Error:       %s\n", vm.ErrorMessage)
	}
	if vm.Host != "" {
		fmt.Printf("Host:        %s\n", vm.Host)
	}
	fmt.Printf("Created:     %s\n", vm.CreatedAt.Format(time.RFC3339))
}

func init() {
	vmListCmd.Flags().Int("page", 1, "Page number")
	vmListCmd.Flags().Int("page-size", 20, "Items per page")
	vmListCmd.Flags().String("status", "", "Filter by status (pending, running, stopped, error, …)")

	vmCreateCmd.Flags().String("name", "", "VM name")
	vmCreateCmd.Flags().String("description", "", "VM description")
	vmCreateCmd.Flags().Int("vcpus", 1, "Number of vCPUs (1-32)")
	vmCreateCmd.Flags().Int("memory", 512, "Memory in MB (128-32768)")
	vmCreateCmd.Flags().String("kernel-path", "", "Path to kernel on the agent host (optional)")
	vmCreateCmd.Flags().String("rootfs-path", "", "Path to rootfs image on the agent host (optional)")
	vmCreateCmd.Flags().String("image-ref", "", "Container image reference to use as rootfs (optional)")
	vmCreateCmd.Flags().Bool("wait", false, "Wait until the VM is running")
	vmCreateCmd.MarkFlagRequired("name")

	vmUpdateCmd.Flags().String("name", "", "New VM name")
	vmUpdateCmd.Flags().String("description", "", "New VM description")

	vmDeployCmd.Flags().String("name", "", "VM name")
	vmDeployCmd.Flags().String("description", "", "VM description")
	vmDeployCmd.Flags().Int("vcpus", 2, "Number of vCPUs (1-32)")
	vmDeployCmd.Flags().Int("memory", 1024, "Memory in MB (128-32768)")
	vmDeployCmd.Flags().String("repo", "", "Public Git repository URL to build and deploy")
	vmDeployCmd.Flags().String("builder", "", "Buildpack builder image (default: paketobuildpacks/builder:base)")
	vmDeployCmd.Flags().String("kernel-path", "", "Path to the kernel on the firecracker-agent host (optional)")
	vmDeployCmd.Flags().Bool("wait", false, "Wait until the VM is running (build + provisioning can take several minutes)")
	vmDeployCmd.MarkFlagRequired("name")
	vmDeployCmd.MarkFlagRequired("repo")

	vmStartCmd.Flags().Bool("wait", false, "Wait until the VM is running")
	vmStopCmd.Flags().Bool("wait", false, "Wait until the VM is stopped")
	vmRestartCmd.Flags().Bool("wait", false, "Wait until the VM is running again")
	vmDeleteCmd.Flags().Bool("wait", false, "Wait until the VM is fully deleted")

	vmCmd.AddCommand(vmListCmd)
	vmCmd.AddCommand(vmGetCmd)
	vmCmd.AddCommand(vmCreateCmd)
	vmCmd.AddCommand(vmUpdateCmd)
	vmCmd.AddCommand(vmDeployCmd)
	vmCmd.AddCommand(vmDeleteCmd)
	vmCmd.AddCommand(vmStartCmd)
	vmCmd.AddCommand(vmStopCmd)
	vmCmd.AddCommand(vmRestartCmd)
}
