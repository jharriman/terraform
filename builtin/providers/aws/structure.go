package aws

import (
	"sort"
	"strconv"
	"strings"

	"github.com/mitchellh/goamz/autoscaling"
	"github.com/mitchellh/goamz/ec2"
	"github.com/mitchellh/goamz/elb"
)

// Takes the result of flatmap.Expand for an array of listeners and
// returns ELB API compatible objects
func expandListeners(configured []interface{}) ([]elb.Listener, error) {
	listeners := make([]elb.Listener, 0, len(configured))

	// Loop over our configured listeners and create
	// an array of goamz compatabile objects
	for _, listener := range configured {
		newL := listener.(map[string]interface{})

		instancePort, err := strconv.ParseInt(newL["instance_port"].(string), 0, 0)
		lbPort, err := strconv.ParseInt(newL["lb_port"].(string), 0, 0)

		if err != nil {
			return nil, err
		}

		l := elb.Listener{
			InstancePort:     instancePort,
			InstanceProtocol: newL["instance_protocol"].(string),
			LoadBalancerPort: lbPort,
			Protocol:         newL["lb_protocol"].(string),
		}

		listeners = append(listeners, l)
	}

	return listeners, nil
}

type ipPermKey struct {
	Protocol string
	FromPort int
	ToPort   int
}

type sortableGroups []ec2.UserSecurityGroup

func (g sortableGroups) Len() int           { return len(g) }
func (g sortableGroups) Swap(i, j int)      { g[i], g[j] = g[j], g[i] }
func (g sortableGroups) Less(i, j int) bool { return g[i].Id < g[j].Id }

type sortableHosts []string

func (g sortableHosts) Len() int           { return len(g) }
func (g sortableHosts) Swap(i, j int)      { g[i], g[j] = g[j], g[i] }
func (g sortableHosts) Less(i, j int) bool { return g[i] < g[j] }

type sortableRules []ec2.IPPerm

func (s sortableRules) Len() int      { return len(s) }
func (s sortableRules) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s sortableRules) Less(i, j int) bool {
	return s[i].Protocol < s[j].Protocol || s[i].FromPort < s[j].FromPort || s[i].ToPort < s[j].ToPort
}

func canonicaliseIPPerms(input []ec2.IPPerm) []ec2.IPPerm {
	rules := map[ipPermKey]*ec2.IPPerm{}
	for _, p := range input {
		key := ipPermKey{p.Protocol, p.FromPort, p.ToPort}
		rule, ok := rules[key]
		if !ok {
			rule = &ec2.IPPerm{
				Protocol: p.Protocol,
				FromPort: p.FromPort,
				ToPort:   p.ToPort,
			}

			rules[key] = rule
		}

		if len(p.SourceGroups) > 0 {
			rule.SourceGroups = append(rule.SourceGroups, p.SourceGroups...)
		}

		if len(p.SourceIPs) > 0 {
			rule.SourceIPs = append(rule.SourceIPs, p.SourceIPs...)
		}
	}

	res := make([]ec2.IPPerm, 0, len(rules))

	for _, r := range rules {
		sort.Sort(sortableHosts(r.SourceIPs))
		sort.Sort(sortableGroups(r.SourceGroups))

		res = append(res, *r)
	}

	sort.Sort(sortableRules(res))

	return res
}

// Takes the result of flatmap.Expand for an array of ingress/egress
// security group rules and returns EC2 API compatible objects
func expandIPPerms(configured []interface{}) ([]ec2.IPPerm, error) {
	perms := make([]ec2.IPPerm, 0, len(configured))

	// Loop over our configured permissions and create
	// an array of goamz/ec2 compatabile objects
	for _, perm := range configured {
		// Our permission object
		newP := perm.(map[string]interface{})

		// Our new returned goamz compatible permission
		p := ec2.IPPerm{}

		// Ports
		if attr, ok := newP["from_port"].(string); ok {
			fromPort, err := strconv.Atoi(attr)
			if err != nil {
				return nil, err
			}
			p.FromPort = fromPort
		} else if attr, ok := newP["from_port"].(int); ok {
			p.FromPort = attr
		}

		if attr, ok := newP["to_port"].(string); ok {
			toPort, err := strconv.Atoi(attr)
			if err != nil {
				return nil, err
			}
			p.ToPort = toPort
		} else if attr, ok := newP["to_port"].(int); ok {
			p.ToPort = attr
		}

		if attr, ok := newP["protocol"].(string); ok {
			p.Protocol = attr
		}

		// Loop over the array of sg ids and built
		// compatibile goamz objects
		if secGroups, ok := newP["security_groups"].([]interface{}); ok {
			expandedGroups := []ec2.UserSecurityGroup{}
			gs := expandStringList(secGroups)

			for _, g := range gs {
				newG := ec2.UserSecurityGroup{
					Id: g,
				}
				expandedGroups = append(expandedGroups, newG)
			}

			p.SourceGroups = expandedGroups
		}

		// Expand CIDR blocks
		if cidrBlocks, ok := newP["cidr_blocks"].([]interface{}); ok {
			p.SourceIPs = expandStringList(cidrBlocks)
		}

		perms = append(perms, p)
	}

	return perms, nil
}

// Flattens an array of ipPerms into a list of primitives that
// flatmap.Flatten() can handle
func flattenIPPerms(list []ec2.IPPerm) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(list))

	for _, perm := range list {
		n := make(map[string]interface{})
		n["from_port"] = perm.FromPort
		n["protocol"] = perm.Protocol
		n["to_port"] = perm.ToPort

		if len(perm.SourceIPs) > 0 {
			n["cidr_blocks"] = perm.SourceIPs
		}

		if v := flattenSecurityGroups(perm.SourceGroups); len(v) > 0 {
			n["security_groups"] = v
		}

		result = append(result, n)
	}

	return result
}

// Flattens a health check into something that flatmap.Flatten()
// can handle
func flattenHealthCheck(check elb.HealthCheck) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, 1)

	chk := make(map[string]interface{})
	chk["unhealthy_threshold"] = int(check.UnhealthyThreshold)
	chk["healthy_threshold"] = int(check.HealthyThreshold)
	chk["target"] = check.Target
	chk["timeout"] = int(check.Timeout)
	chk["interval"] = int(check.Interval)

	result = append(result, chk)

	return result
}

// Flattens an array of UserSecurityGroups into a []string
func flattenSecurityGroups(list []ec2.UserSecurityGroup) []string {
	result := make([]string, 0, len(list))
	for _, g := range list {
		result = append(result, g.Id)
	}
	return result
}

// Flattens an array of SecurityGroups into a []string
func flattenAutoscalingSecurityGroups(list []autoscaling.SecurityGroup) []string {
	result := make([]string, 0, len(list))
	for _, g := range list {
		result = append(result, g.SecurityGroup)
	}
	return result
}

// Flattens an array of AvailabilityZones into a []string
func flattenAvailabilityZones(list []autoscaling.AvailabilityZone) []string {
	result := make([]string, 0, len(list))
	for _, g := range list {
		result = append(result, g.AvailabilityZone)
	}
	return result
}

// Flattens an array of LoadBalancerName into a []string
func flattenLoadBalancers(list []autoscaling.LoadBalancerName) []string {
	result := make([]string, 0, len(list))
	for _, g := range list {
		if g.LoadBalancerName != "" {
			result = append(result, g.LoadBalancerName)
		}
	}
	return result
}

// Flattens an array of Instances into a []string
func flattenInstances(list []elb.Instance) []string {
	result := make([]string, 0, len(list))
	for _, i := range list {
		result = append(result, i.InstanceId)
	}
	return result
}

// Takes the result of flatmap.Expand for an array of strings
// and returns a []string
func expandStringList(configured []interface{}) []string {
	// here we special case the * expanded lists. For example:
	//
	//	 instances = ["${aws_instance.foo.*.id}"]
	//
	if len(configured) == 1 && strings.Contains(configured[0].(string), ",") {
		return strings.Split(configured[0].(string), ",")
	}

	vs := make([]string, 0, len(configured))
	for _, v := range configured {
		vs = append(vs, v.(string))
	}
	return vs
}
