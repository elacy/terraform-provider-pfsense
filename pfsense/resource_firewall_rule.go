package pfsense

import (
	"context"
	"slices"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/sjafferali/pfsense-api-goclient/pfsenseapi"
)

func resourceFirewallRule() *resource[pfsenseapi.FirewallRuleRequest, pfsenseapi.FirewallRule, int] {
	return &resource[pfsenseapi.FirewallRuleRequest, pfsenseapi.FirewallRule, int]{
		name:        "pfsense_firewall_rule",
		description: "Firewall Rule",
		delete: func(ctx context.Context, client *pfsenseapi.Client, _ string, id int) error {
			return client.Firewall.DeleteRule(ctx, id, true)
		},
		list: func(ctx context.Context, client *pfsenseapi.Client, _ string) ([]*pfsenseapi.FirewallRule, error) {
			return client.Firewall.ListRules(ctx)
		},
		update: func(ctx context.Context, client *pfsenseapi.Client, id int, request *pfsenseapi.FirewallRuleRequest) (*pfsenseapi.FirewallRule, error) {
			return client.Firewall.UpdateRule(ctx, id, *request, true)
		},
		create: func(ctx context.Context, client *pfsenseapi.Client, request *pfsenseapi.FirewallRuleRequest) (*pfsenseapi.FirewallRule, error) {
			return client.Firewall.CreateRule(ctx, *request, true)
		},
		getId: func(_ context.Context, _ *pfsenseapi.Client, response *pfsenseapi.FirewallRule) (int, error) {
			return int(response.Tracker), nil
		},
		properties: map[string]*resourceProperty[pfsenseapi.FirewallRuleRequest, pfsenseapi.FirewallRule]{
			"ack_queue": {
				schema: &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Acknowledge traffic shaper queue to apply to this rule. This must be an existing traffic shaper queue and cannot match the `defaultqueue` value.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.FirewallRuleRequest) error {
					req.AckQueue = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(res *pfsenseapi.FirewallRule) (interface{}, error) {
					return res.AckQueue, nil
				},
			},
			"default_queue": {
				schema: &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Default traffic shaper queue to apply to this rule. This must be an existing traffic shaper queue name. This field is required when an `ackqueue` value is provided.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.FirewallRuleRequest) error {
					req.DefaultQueue = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(res *pfsenseapi.FirewallRule) (interface{}, error) {
					return res.DefaultQueue, nil
				},
			},
			"description": {
				schema: &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Description for the rule.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.FirewallRuleRequest) error {
					req.Descr = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(res *pfsenseapi.FirewallRule) (interface{}, error) {
					return res.Descr, nil
				},
			},
			"direction": {
				schema: &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					Default:      "any",
					Description:  "Direction of floating firewall rule. This parameter is only avilable when `floating` is set to `true`.",
					ValidateFunc: validation.StringInSlice([]string{"in", "out", "any"}, false),
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.FirewallRuleRequest) error {
					req.Direction = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(res *pfsenseapi.FirewallRule) (interface{}, error) {
					return res.Direction, nil
				},
			},
			"disabled": {
				schema: &schema.Schema{
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     false,
					Description: "Disable the rule.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.FirewallRuleRequest) error {
					req.Disabled = d.Get(name).(bool)
					return nil
				},
				getFromResponse: func(res *pfsenseapi.FirewallRule) (interface{}, error) {
					return res.Disabled, nil
				},
			},
			"dn_pipe": {
				schema: &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Traffic shaper limiter (in) queue for this rule. This must be an existing traffic shaper limiter or queue. This field is required if a `pdnpipe` value is provided.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.FirewallRuleRequest) error {
					req.DNPipe = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(res *pfsenseapi.FirewallRule) (interface{}, error) {
					return res.Dnpipe, nil
				},
			},
			"destination": {
				schema: &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "any",
					Description: "Destination address of the firewall rule. This may be a single IP, network CIDR, alias name, or interface. When specifying an interface, you may use the real interface ID (e.g. igb0), the descriptive interface name, or the pfSense ID (e.g. wan, lan, optx). To use only the  interface's assigned address, add `ip` to the end of the interface name otherwise  the entire interface's subnet is implied. To negate the context of the destination address, you may prefix the value with `!`.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.FirewallRuleRequest) error {
					req.Dst = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(res *pfsenseapi.FirewallRule) (interface{}, error) {
					if res.Destination == nil {
						return nil, nil
					}

					return res.Destination.TargetString(), nil
				},
			},
			"destination_port": {
				schema: &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "any",
					Description: "TCP and/or UDP destination port, port range or port alias to apply to this rule. You may specify `any` to match any destination port. This parameter is required when `protocol` is set to `tcp`, `udp`, or `tcp/udp`.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.FirewallRuleRequest) error {
					req.DstPort = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(res *pfsenseapi.FirewallRule) (interface{}, error) {
					if res.Destination == nil {
						return nil, nil
					}
					return res.Destination.Port, nil
				},
			},
			"floating": {
				schema: &schema.Schema{
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     false,
					Description: "Set this rule as a floating firewall rule.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.FirewallRuleRequest) error {
					req.Floating = d.Get(name).(bool)
					return nil
				},
				getFromResponse: func(res *pfsenseapi.FirewallRule) (interface{}, error) {
					return res.Floating == "yes", nil
				},
			},
			"gateway": {
				schema: &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					Description:  "Name of an existing gateway traffic will route over upon match. Do not specify this parameter to assume the default gateway. The gateway specified must be of the same IP type set in `ipprotocol`.",
					ValidateFunc: validation.IsIPAddress,
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.FirewallRuleRequest) error {
					req.Gateway = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(res *pfsenseapi.FirewallRule) (interface{}, error) {
					return res.Gateway, nil
				},
			},
			"icmp_type": {
				schema: &schema.Schema{
					Type:        schema.TypeList,
					Optional:    true,
					Description: "ICMP subtypes of the firewall rule. This parameter is only available when `protocol` is set to `icmp`. If this parameter is not specified, all ICMP subtypes will be assumed.",
					Elem: &schema.Schema{
						Type:         schema.TypeString,
						ValidateFunc: validation.StringInSlice([]string{"althost", "dataconv", "echorep", "echoreq", "inforep", "inforeq", "ipv6-here", "ipv6-where", "maskrep", "maskreq", "mobredir", "mobregrep", "mobregreq", "paramprob", "photuris", "redir", "routeradv", "routersol", "skip", "squench", "timerep", "timereq", "timex", "trace", "unreach"}, false),
					},
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.FirewallRuleRequest) error {
					result, err := interfaceToStringArray(d.Get(name))

					if err != nil {
						return err
					}

					req.ICMPType = result
					return nil
				},
				getFromResponse: func(res *pfsenseapi.FirewallRule) (interface{}, error) {
					return splitIntoArray(res.ICMPType, ","), nil
				},
			},
			"interface": {
				schema: &schema.Schema{
					Type:        schema.TypeList,
					Required:    true,
					Description: "Interface this rule will apply to. You may specify either the interface's descriptive name, the pfSense  interface ID (e.g. wan, lan, optx), or the real interface ID (e.g. igb0). If `floating` is enabled, multiple interfaces may be specified.",
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.FirewallRuleRequest) error {
					result, err := interfaceToStringArray(d.Get(name))

					if err != nil {
						return err
					}

					req.Interface = result
					return nil
				},
				getFromResponse: func(res *pfsenseapi.FirewallRule) (interface{}, error) {
					return splitIntoArray(res.Interface, ","), nil
				},
			},
			"ip_protocol": {
				schema: &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					Default:      "inet",
					Description:  "IP protocol(s) this rule will apply to.",
					ValidateFunc: validation.StringInSlice([]string{"inet", "inet6", "inet46"}, false),
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.FirewallRuleRequest) error {
					req.IPProtocol = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(res *pfsenseapi.FirewallRule) (interface{}, error) {
					return res.IPProtocol, nil
				},
			},
			"log": {
				schema: &schema.Schema{
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     false,
					Description: "Enable logging of traffic matching this rule.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.FirewallRuleRequest) error {
					req.Log = d.Get(name).(bool)
					return nil
				},
				getFromResponse: func(res *pfsenseapi.FirewallRule) (interface{}, error) {
					return res.Log, nil
				},
			},
			"pdn_pipe": {
				schema: &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Traffic shaper limiter (out) queue for this rule. This must be an existing traffic shaper limiter or queue. This value cannot match the `dnpipe` value and must be a child queue if `dnpipe` is a child queue, or a parent limiter if `dnpipe` is a parent limiter.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.FirewallRuleRequest) error {
					req.PDNPipe = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(res *pfsenseapi.FirewallRule) (interface{}, error) {
					return res.PDNPipe, nil
				},
			},
			"protocol": {
				schema: &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					Default:      "any",
					Description:  "Transfer protocol this rule will apply to.",
					ValidateFunc: validation.StringInSlice([]string{"any", "tcp", "udp", "tcp/udp", "icmp", "esp", "ah", "gre", "ipv6", "igmp", "pim", "ospf", "carp", "pfsync"}, false),
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.FirewallRuleRequest) error {
					req.Protocol = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(res *pfsenseapi.FirewallRule) (interface{}, error) {
					return res.Protocol, nil
				},
			},
			"quick": {
				schema: &schema.Schema{
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     false,
					Description: "Apply action immediately upon match. This field is only available for `floating` rules.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.FirewallRuleRequest) error {
					req.Quick = d.Get(name).(bool)
					return nil
				},
				getFromResponse: func(res *pfsenseapi.FirewallRule) (interface{}, error) {
					return res.Quick == "yes", nil
				},
			},
			"schedule": {
				schema: &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Firewall schedule to apply to this rule. This must be an existing firewall schedule name.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.FirewallRuleRequest) error {
					req.Sched = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(res *pfsenseapi.FirewallRule) (interface{}, error) {
					return res.Sched, nil
				},
			},
			"source": {
				schema: &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "any",
					Description: "Source address of the firewall rule. This may be a single IP, network CIDR, alias name, or interface. When specifying an interface, you may use the real interface ID (e.g. igb0), the descriptive interface name, or the pfSense ID (e.g. wan, lan, optx). To use only the  interface's assigned address, add `ip` to the end of the interface name otherwise  the entire interface's subnet is implied. To negate the context of the source address, you may prefix the value with `!`.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.FirewallRuleRequest) error {
					req.Src = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(res *pfsenseapi.FirewallRule) (interface{}, error) {
					if res.Source == nil {
						return nil, nil
					}

					return res.Source.TargetString(), nil
				},
			},
			"source_port": {
				schema: &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "any",
					Description: "TCP and/or UDP source port, port range or port alias  to apply to this rule. You may specify `any` to match any source port. This parameter is required when `protocol` is set to `tcp`, `udp`, or `tcp/udp`.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.FirewallRuleRequest) error {
					req.SrcPort = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(res *pfsenseapi.FirewallRule) (interface{}, error) {
					if res.Source == nil {
						return nil, nil
					}

					return res.Source.Port, nil
				},
			},
			"state_type": {
				schema: &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					Description:  "State type to use when this rule is matched.",
					ValidateFunc: validation.StringInSlice([]string{"keep state", "sloppy state", "synproxy state"}, false),
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.FirewallRuleRequest) error {
					req.StateType = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(res *pfsenseapi.FirewallRule) (interface{}, error) {
					return res.Statetype, nil
				},
			},
			"tcp_flag": {
				schema: &schema.Schema{
					Type:        schema.TypeList,
					Optional:    true,
					Description: "Use this to choose TCP flags that must be set or cleared for this rule to match.",
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"flag": {
								Type:         schema.TypeString,
								ValidateFunc: validation.StringInSlice([]string{"fin", "syn", "rst", "psh", "ack", "urg", "ece", "cwr"}, false),
								Required:     true,
							},
							"present": {
								Type:     schema.TypeBool,
								Required: true,
							},
						},
					},
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.FirewallRuleRequest) error {
					flags := d.Get("tcp_flag").([]interface{})

					var setFlags []string
					var coveredFlags []string

					for _, flag := range flags {
						flagMap := flag.(map[string]interface{})
						flagName := flagMap["flag"].(string)

						coveredFlags = append(coveredFlags, flagName)

						if flagMap["present"] == true {
							setFlags = append(setFlags, flagName)
						}
					}

					req.TCPFlags1 = setFlags
					req.TCPFlags2 = coveredFlags

					if len(req.TCPFlags2) == 0 {
						req.TCPFlagsAny = true
					}

					return nil
				},
				getFromResponse: func(res *pfsenseapi.FirewallRule) (interface{}, error) {
					flags := []interface{}{}
					setFlags := splitIntoArray(res.TCPFlags1, ",")

					for _, flag := range splitIntoArray(res.TCPFlags2, ",") {
						flags = append(flags, map[string]interface{}{
							"flag":    flag,
							"present": slices.Contains(setFlags, flag),
						})
					}

					return flags, nil
				},
			},
			"type": {
				schema: &schema.Schema{
					Type:         schema.TypeString,
					Required:     true,
					Description:  "Firewall rule type.",
					ValidateFunc: validation.StringInSlice([]string{"pass", "block", "reject"}, false),
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.FirewallRuleRequest) error {
					req.Type = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(res *pfsenseapi.FirewallRule) (interface{}, error) {
					return res.Type, nil
				},
			},
		},
	}
}
