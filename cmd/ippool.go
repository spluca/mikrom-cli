package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spluca/mikrom-cli/internal/api"
)

var ippoolCmd = &cobra.Command{
	Use:   "ippool",
	Short: "Manage IP pools",
}

var ippoolListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all IP pools",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		page, _ := cmd.Flags().GetInt("page")
		pageSize, _ := cmd.Flags().GetInt("page-size")

		resp, err := newClient().ListIPPools(page, pageSize)
		if err != nil {
			return err
		}

		if len(resp.IPPools) == 0 {
			fmt.Println("No IP pools found")
			return nil
		}

		fmt.Printf("%-36s  %-20s  %-18s  %-15s  %-15s\n", "ID", "NAME", "CIDR", "START", "END")
		for _, p := range resp.IPPools {
			fmt.Printf("%-36s  %-20s  %-18s  %-15s  %-15s\n",
				p.ID, p.Name, p.CIDR, p.StartIP, p.EndIP)
		}
		fmt.Printf("\nTotal: %d\n", resp.Total)
		return nil
	},
}

var ippoolGetCmd = &cobra.Command{
	Use:   "get <pool-id>",
	Short: "Get IP pool details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()

		pool, err := newClient().GetIPPool(args[0])
		if err != nil {
			return err
		}

		printIPPool(pool)
		return nil
	},
}

var ippoolCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new IP pool",
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()
		name, _ := cmd.Flags().GetString("name")
		cidr, _ := cmd.Flags().GetString("cidr")
		gateway, _ := cmd.Flags().GetString("gateway")
		startIP, _ := cmd.Flags().GetString("start-ip")
		endIP, _ := cmd.Flags().GetString("end-ip")

		pool, err := newClient().CreateIPPool(api.CreateIPPoolRequest{
			Name:    name,
			CIDR:    cidr,
			Gateway: gateway,
			StartIP: startIP,
			EndIP:   endIP,
		})
		if err != nil {
			return err
		}

		fmt.Printf("IP pool created: %s\n", pool.ID)
		printIPPool(pool)
		return nil
	},
}

var ippoolDeleteCmd = &cobra.Command{
	Use:   "delete <pool-id>",
	Short: "Delete an IP pool",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()

		if err := newClient().DeleteIPPool(args[0]); err != nil {
			return err
		}

		fmt.Printf("IP pool %s deleted\n", args[0])
		return nil
	},
}

var ippoolStatsCmd = &cobra.Command{
	Use:   "stats <pool-id>",
	Short: "Show IP allocation stats for a pool",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAuth()

		stats, err := newClient().GetIPPoolStats(args[0])
		if err != nil {
			return err
		}

		fmt.Printf("Total:     %d\n", stats.Total)
		fmt.Printf("Allocated: %d\n", stats.Allocated)
		fmt.Printf("Available: %d\n", stats.Available)
		return nil
	},
}

func printIPPool(p *api.IPPool) {
	fmt.Printf("ID:      %s\n", p.ID)
	fmt.Printf("Name:    %s\n", p.Name)
	fmt.Printf("CIDR:    %s\n", p.CIDR)
	fmt.Printf("Gateway: %s\n", p.Gateway)
	fmt.Printf("Start:   %s\n", p.StartIP)
	fmt.Printf("End:     %s\n", p.EndIP)
}

func init() {
	ippoolListCmd.Flags().Int("page", 1, "Page number")
	ippoolListCmd.Flags().Int("page-size", 20, "Items per page")

	ippoolCreateCmd.Flags().String("name", "", "Pool name")
	ippoolCreateCmd.Flags().String("cidr", "", "Network CIDR (e.g. 10.100.0.0/24)")
	ippoolCreateCmd.Flags().String("gateway", "", "Gateway IP")
	ippoolCreateCmd.Flags().String("start-ip", "", "First usable IP")
	ippoolCreateCmd.Flags().String("end-ip", "", "Last usable IP")
	ippoolCreateCmd.MarkFlagRequired("name")
	ippoolCreateCmd.MarkFlagRequired("cidr")
	ippoolCreateCmd.MarkFlagRequired("gateway")
	ippoolCreateCmd.MarkFlagRequired("start-ip")
	ippoolCreateCmd.MarkFlagRequired("end-ip")

	ippoolCmd.AddCommand(ippoolListCmd)
	ippoolCmd.AddCommand(ippoolGetCmd)
	ippoolCmd.AddCommand(ippoolCreateCmd)
	ippoolCmd.AddCommand(ippoolDeleteCmd)
	ippoolCmd.AddCommand(ippoolStatsCmd)
}
