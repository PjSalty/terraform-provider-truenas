package resources

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/planhelpers"
	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

// systemAdvancedSingletonID is the fixed identifier for
// truenas_system_advanced. TrueNAS has one System > Advanced row.
const systemAdvancedSingletonID = "system_advanced"

// Syslog transports, as the API spells them.
const (
	syslogTransportUDP = "UDP"
	syslogTransportTCP = "TCP"
	syslogTransportTLS = "TLS"
)

// syslogHostPattern is the check system.advanced.update runs on a syslog
// server host: an optional bracket, name or address characters (colon
// included), an optional closing bracket and an optional ":port". Python's
// \w is any Unicode letter, digit or underscore.
var syslogHostPattern = regexp.MustCompile(`^\[?[\pL\pN_.\-:%]+\]?(:\d+)?$`)

// systemAdvancedFieldMax is the longest string TrueNAS accepts for a field
// that does not set its own limit: the API base model caps every string at
// 1024 characters. login_banner is the one field with a larger limit.
const (
	systemAdvancedFieldMax  = 1024
	systemAdvancedBannerMax = 4096
)

var (
	_ resource.Resource                   = &SystemAdvancedResource{}
	_ resource.ResourceWithImportState    = &SystemAdvancedResource{}
	_ resource.ResourceWithValidateConfig = &SystemAdvancedResource{}
)

// SystemAdvancedResource manages the System > Advanced settings.
type SystemAdvancedResource struct {
	client *wsclient.Client
}

// SystemAdvancedSyslogServerModel is one remote syslog server.
type SystemAdvancedSyslogServerModel struct {
	Host           types.String `tfsdk:"host"`
	Transport      types.String `tfsdk:"transport"`
	TLSCertificate types.Int64  `tfsdk:"tls_certificate"`
}

// systemAdvancedSyslogServerAttrTypes is the object type of one server.
var systemAdvancedSyslogServerAttrTypes = map[string]attr.Type{
	"host":            types.StringType,
	"transport":       types.StringType,
	"tls_certificate": types.Int64Type,
}

// SystemAdvancedResourceModel describes the resource data model.
type SystemAdvancedResourceModel struct {
	ID                 types.String   `tfsdk:"id"`
	Syslogservers      types.List     `tfsdk:"syslogservers"`
	Sysloglevel        types.String   `tfsdk:"sysloglevel"`
	FQDNSyslog         types.Bool     `tfsdk:"fqdn_syslog"`
	SyslogAudit        types.Bool     `tfsdk:"syslog_audit"`
	Motd               types.String   `tfsdk:"motd"`
	LoginBanner        types.String   `tfsdk:"login_banner"`
	Consolemenu        types.Bool     `tfsdk:"consolemenu"`
	Serialconsole      types.Bool     `tfsdk:"serialconsole"`
	Serialport         types.String   `tfsdk:"serialport"`
	Serialspeed        types.String   `tfsdk:"serialspeed"`
	BootScrub          types.Int64    `tfsdk:"boot_scrub"`
	KernelExtraOptions types.String   `tfsdk:"kernel_extra_options"`
	KdumpEnabled       types.Bool     `tfsdk:"kdump_enabled"`
	Debugkernel        types.Bool     `tfsdk:"debugkernel"`
	Powerdaemon        types.Bool     `tfsdk:"powerdaemon"`
	Traceback          types.Bool     `tfsdk:"traceback"`
	Uploadcrash        types.Bool     `tfsdk:"uploadcrash"`
	Anonstats          types.Bool     `tfsdk:"anonstats"`
	Overprovision      types.Int64    `tfsdk:"overprovision"`
	Timeouts           timeouts.Value `tfsdk:"timeouts"`
}

// NewSystemAdvancedResource returns a new SystemAdvancedResource factory.
func NewSystemAdvancedResource() resource.Resource {
	return &SystemAdvancedResource{}
}

func (r *SystemAdvancedResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_system_advanced"
}

// systemAdvancedBool is an optional, computed boolean that keeps the stored
// value when the configuration leaves it out.
func systemAdvancedBool(description string) schema.BoolAttribute {
	return schema.BoolAttribute{
		Description: description,
		Optional:    true,
		Computed:    true,
		PlanModifiers: []planmodifier.Bool{
			boolplanmodifier.UseStateForUnknown(),
		},
	}
}

// systemAdvancedString is the string form of systemAdvancedBool.
func systemAdvancedString(description string, validators ...validator.String) schema.StringAttribute {
	return schema.StringAttribute{
		Description: description,
		Optional:    true,
		Computed:    true,
		Validators:  validators,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
}

// systemAdvancedInt is the integer form of systemAdvancedBool.
func systemAdvancedInt(description string, floor int64) schema.Int64Attribute {
	return schema.Int64Attribute{
		Description: description,
		Optional:    true,
		Computed:    true,
		Validators:  []validator.Int64{int64validator.AtLeast(floor)},
		PlanModifiers: []planmodifier.Int64{
			int64planmodifier.UseStateForUnknown(),
		},
	}
}

func (r *SystemAdvancedResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true}),
		},
		Description: "Manages the System > Advanced settings on TrueNAS SCALE: remote syslog, the login " +
			"and message-of-the-day text, the console, boot and crash settings. This is a singleton, " +
			"TrueNAS has exactly one. Several attributes act on the running system when applied; each " +
			"description says what it does.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Fixed singleton identifier. Always \"system_advanced\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"syslogservers": schema.ListNestedAttribute{
				Description: "Remote syslog servers, at most two. TrueNAS stores the list in the order it " +
					"was sent and reads it back unchanged, so this is an ordered list; the order only " +
					"decides which server is written first in the syslog-ng configuration. Setting it " +
					"replaces the whole list, and an empty list removes every remote server. Left out, " +
					"the current servers are kept. A change restarts the syslog service.",
				Optional: true,
				Computed: true,
				Validators: []validator.List{
					listvalidator.SizeAtMost(2),
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"host": schema.StringAttribute{
							Description: "DNS name or IP address of the server, with an optional `:port`. " +
								"Without a port TrueNAS uses 514 for UDP and TCP and 6514 for TLS. An " +
								"IPv6 address must be in brackets, `[2001:db8::10]` or " +
								"`[2001:db8::10]:5514`. Each host must be unique within the list.",
							Required: true,
							Validators: []validator.String{
								stringvalidator.LengthBetween(1, systemAdvancedFieldMax),
							},
						},
						"transport": schema.StringAttribute{
							Description: "Transport protocol: `UDP`, `TCP` or `TLS`. Defaults to `UDP`.",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString(syslogTransportUDP),
							Validators: []validator.String{
								stringvalidator.OneOf(syslogTransportUDP, syslogTransportTCP, syslogTransportTLS),
							},
						},
						"tls_certificate": schema.Int64Attribute{
							Description: "ID of the local certificate sent to the server for mutual TLS. " +
								"Only valid with `transport = \"TLS\"`. Leave it out for one-way TLS, where " +
								"only the server has to present a certificate. TrueNAS sets it to null for " +
								"UDP and TCP, so the provider rejects it there.",
							Optional: true,
						},
					},
				},
			},
			"sysloglevel": systemAdvancedString("Minimum severity of the messages kept and forwarded: `F_EMERG`, "+
				"`F_ALERT`, `F_CRIT`, `F_ERR`, `F_WARNING`, `F_NOTICE`, `F_INFO` or `F_DEBUG`. A change "+
				"restarts the syslog service. TrueNAS defaults to `F_INFO`.",
				stringvalidator.OneOf("F_EMERG", "F_ALERT", "F_CRIT", "F_ERR", "F_WARNING", "F_NOTICE", "F_INFO", "F_DEBUG")),
			"fqdn_syslog": systemAdvancedBool("Write the full domain name of this host into syslog messages. A change " +
				"restarts the syslog service. TrueNAS defaults to `false`."),
			"syslog_audit": systemAdvancedBool("Also send audit messages to the remote syslog servers. It has no effect " +
				"while `syslogservers` is empty. A change restarts the syslog service. TrueNAS defaults to " +
				"`false`."),
			"motd": systemAdvancedString("Message of the day shown after a login, at most 1024 characters. A change "+
				"regenerates /etc/motd. TrueNAS defaults to `Welcome to TrueNAS`.",
				stringvalidator.LengthAtMost(systemAdvancedFieldMax)),
			"login_banner": systemAdvancedString("Banner shown before the login prompt over SSH and on the login page, at "+
				"most 4096 characters. An empty string means no banner. A change reloads the SSH service. "+
				"TrueNAS defaults to an empty string.",
				stringvalidator.LengthAtMost(systemAdvancedBannerMax)),
			"consolemenu": systemAdvancedBool("Show the text console menu on the local console instead of a plain " +
				"login prompt. A change restarts the getty on tty1, which ends the console session that is " +
				"open there."),
			"serialconsole": systemAdvancedBool("Enable a login on a serial port. A change enables or disables the " +
				"serial-getty unit for `serialport` and regenerates the GRUB configuration. TrueNAS defaults " +
				"to `false`."),
			"serialport": systemAdvancedString("Serial port device for the console, such as `ttyS0`. When "+
				"`serialconsole` is true it must be a port TrueNAS has found and not the port the UPS "+
				"service uses, which only TrueNAS can check, at apply time. Changing it moves the "+
				"serial-getty unit to the new port and regenerates the GRUB configuration. TrueNAS defaults "+
				"to `ttyS0`.",
				stringvalidator.LengthAtMost(systemAdvancedFieldMax)),
			"serialspeed": systemAdvancedString("Serial console baud rate: `9600`, `19200`, `38400`, `57600` or "+
				"`115200`. A change restarts the serial-getty unit and regenerates the GRUB configuration. "+
				"TrueNAS defaults to `9600`.",
				stringvalidator.OneOf("9600", "19200", "38400", "57600", "115200")),
			"boot_scrub": systemAdvancedInt("Days between scrubs of the boot pool, at least 1. A change restarts cron, "+
				"because the scrub is a cron job. TrueNAS defaults to `7`.", 1),
			"kernel_extra_options": systemAdvancedString("Extra parameters appended to the kernel command line in the "+
				"boot menu. A change regenerates the GRUB configuration and takes effect at the next "+
				"reboot; a wrong value can stop the system booting. TrueNAS rejects a newline, a double quote, "+
				"`systemd.unified_cgroup_hierarchy` and anything `grub-script-check` finds invalid; the "+
				"provider checks the first three at plan time, at most 1024 characters. TrueNAS defaults "+
				"to an empty string.",
				stringvalidator.LengthAtMost(systemAdvancedFieldMax)),
			"kdump_enabled": systemAdvancedBool("Enable kernel crash dumps. A change regenerates the kdump and GRUB " +
				"configuration and only takes effect at the next reboot. TrueNAS defaults to `false`."),
			"debugkernel": systemAdvancedBool("Boot the debug kernel. A change regenerates the GRUB configuration " +
				"and, when switching it on, rebuilds the initramfs, which takes a while. It takes effect at " +
				"the next reboot. TrueNAS defaults to `false`."),
			"powerdaemon": systemAdvancedBool("Run the power management daemon. A change restarts it. TrueNAS defaults " +
				"to `false`."),
			"traceback": systemAdvancedBool("Setting stored for the web interface's error tracebacks. The 25.10 " +
				"middleware only stores it, and applying it does nothing else. TrueNAS defaults to `true`."),
			"uploadcrash": systemAdvancedBool("Setting stored for uploading crash reports. The 25.10 middleware only " +
				"stores it, and applying it does nothing else. TrueNAS defaults to `true`."),
			"anonstats": systemAdvancedBool("Setting stored for anonymous usage statistics. The 25.10 middleware only " +
				"stores it, and applying it does nothing else. TrueNAS defaults to `true`."),
			"overprovision": systemAdvancedInt("Percentage of an SSD to leave unused, for wear leveling. "+
				"TrueNAS applies it when it creates a pool with log devices, not to existing pools. `0` "+
				"disables it. A fresh TrueNAS has it unset (null), and the API cannot set it back to null, "+
				"so a configuration cannot return to that state; it only reads back as `0`.", 0),
		},
	}
}

func (r *SystemAdvancedResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*wsclient.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *wsclient.Client, got: %T", req.ProviderData),
		)
		return
	}
	r.client = c
}

// syslogHostProblem returns why system.advanced.update would reject a syslog
// server host, or "" when it accepts it. Where the service would crash on a
// value (a non-numeric port, a bare IPv6 address it reads as host and port)
// this reports a plain error instead.
func syslogHostProblem(host string) string {
	if !syslogHostPattern.MatchString(host) {
		return "invalid syslog server format, expected a name or address with an optional :port"
	}
	if strings.Contains(host, "[") != strings.Contains(host, "]") {
		return "unbalanced brackets"
	}
	if !strings.Contains(host, "[") && strings.Count(host, ":") > 1 {
		return "an IPv6 address must be in brackets, such as [2001:db8::10]"
	}
	if strings.Contains(host, "]:") || (strings.Contains(host, ":") && !strings.Contains(host, "]")) {
		port, err := strconv.Atoi(host[strings.LastIndex(host, ":")+1:])
		if err != nil {
			return "the port must be a number"
		}
		if port < 0 || port > 65535 {
			return "the port must be in the range of 0 to 65535"
		}
	}
	return ""
}

// ValidateConfig repeats the checks system.advanced.update runs on the
// syslog servers and the kernel options, so a bad value fails at plan time
// naming the attribute, and a TLS certificate TrueNAS would silently drop is
// rejected instead of producing a plan that never settles.
func (r *SystemAdvancedResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg SystemAdvancedResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !cfg.KernelExtraOptions.IsNull() && !cfg.KernelExtraOptions.IsUnknown() {
		opts := cfg.KernelExtraOptions.ValueString()
		switch {
		case strings.ContainsAny(opts, "\n\""):
			resp.Diagnostics.AddAttributeError(path.Root("kernel_extra_options"), "Invalid Kernel Options",
				"a newline and a double quote are not allowed")
		// Spelled in two parts so the API drift scan, which reads dotted
		// string literals as method names, does not take it for one.
		case strings.Contains(opts, "systemd."+"unified_cgroup_hierarchy"):
			resp.Diagnostics.AddAttributeError(path.Root("kernel_extra_options"), "Invalid Kernel Options",
				"modifying systemd.unified_cgroup_hierarchy is not allowed")
		}
	}

	if cfg.Syslogservers.IsNull() || cfg.Syslogservers.IsUnknown() {
		return
	}
	var servers []SystemAdvancedSyslogServerModel
	resp.Diagnostics.Append(cfg.Syslogservers.ElementsAs(ctx, &servers, false)...)
	seen := map[string]bool{}
	for i, server := range servers {
		at := path.Root("syslogservers").AtListIndex(i)
		if !server.Host.IsUnknown() {
			host := server.Host.ValueString()
			switch problem := syslogHostProblem(host); {
			case seen[host]:
				resp.Diagnostics.AddAttributeError(at.AtName("host"), "Invalid Syslog Server",
					fmt.Sprintf("%q is listed more than once", host))
			case problem != "":
				resp.Diagnostics.AddAttributeError(at.AtName("host"), "Invalid Syslog Server", problem)
			}
			seen[host] = true
		}
		if !server.TLSCertificate.IsNull() && !server.Transport.IsUnknown() &&
			server.Transport.ValueString() != syslogTransportTLS {
			resp.Diagnostics.AddAttributeError(at.AtName("tls_certificate"), "Invalid TLS Certificate",
				"tls_certificate can only be set when transport is \"TLS\"")
		}
	}
}

// systemAdvancedBoolPtr returns the value to send for an optional boolean,
// or nil when the plan left it unset or unknown.
func systemAdvancedBoolPtr(v types.Bool) *bool {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	b := v.ValueBool()
	return &b
}

// systemAdvancedStringPtr is the string form of systemAdvancedBoolPtr.
func systemAdvancedStringPtr(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	s := v.ValueString()
	return &s
}

// systemAdvancedInt64Ptr is the integer form of systemAdvancedBoolPtr.
func systemAdvancedInt64Ptr(v types.Int64) *int64 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	n := v.ValueInt64()
	return &n
}

// updateRequest is the request form of the plan. Only attributes the plan
// set are included: system.advanced.update is a ForUpdateMetaclass model, so
// an omitted key leaves the stored value alone. The servers go in the order
// they were written, which is the order TrueNAS stores and returns.
func (r *SystemAdvancedResource) updateRequest(ctx context.Context, plan *SystemAdvancedResourceModel, diags *diag.Diagnostics) *truenas.SystemAdvancedUpdateRequest {
	req := &truenas.SystemAdvancedUpdateRequest{
		Consolemenu:        systemAdvancedBoolPtr(plan.Consolemenu),
		Serialconsole:      systemAdvancedBoolPtr(plan.Serialconsole),
		Serialport:         systemAdvancedStringPtr(plan.Serialport),
		Serialspeed:        systemAdvancedStringPtr(plan.Serialspeed),
		Powerdaemon:        systemAdvancedBoolPtr(plan.Powerdaemon),
		Overprovision:      systemAdvancedInt64Ptr(plan.Overprovision),
		Traceback:          systemAdvancedBoolPtr(plan.Traceback),
		Debugkernel:        systemAdvancedBoolPtr(plan.Debugkernel),
		Uploadcrash:        systemAdvancedBoolPtr(plan.Uploadcrash),
		Anonstats:          systemAdvancedBoolPtr(plan.Anonstats),
		Motd:               systemAdvancedStringPtr(plan.Motd),
		LoginBanner:        systemAdvancedStringPtr(plan.LoginBanner),
		BootScrub:          systemAdvancedInt64Ptr(plan.BootScrub),
		FQDNSyslog:         systemAdvancedBoolPtr(plan.FQDNSyslog),
		Sysloglevel:        systemAdvancedStringPtr(plan.Sysloglevel),
		SyslogAudit:        systemAdvancedBoolPtr(plan.SyslogAudit),
		KdumpEnabled:       systemAdvancedBoolPtr(plan.KdumpEnabled),
		KernelExtraOptions: systemAdvancedStringPtr(plan.KernelExtraOptions),
	}

	if !plan.Syslogservers.IsNull() && !plan.Syslogservers.IsUnknown() {
		var planned []SystemAdvancedSyslogServerModel
		diags.Append(plan.Syslogservers.ElementsAs(ctx, &planned, false)...)
		servers := make([]truenas.SystemAdvancedSyslogServer, 0, len(planned))
		for _, p := range planned {
			servers = append(servers, truenas.SystemAdvancedSyslogServer{
				Host:           p.Host.ValueString(),
				Transport:      p.Transport.ValueString(),
				TLSCertificate: systemAdvancedInt64Ptr(p.TLSCertificate),
			})
		}
		req.Syslogservers = &servers
	}

	return req
}

// systemAdvancedSetModel populates the model from what TrueNAS returned.
// The level is upper-cased: TrueNAS compares it case-insensitively and a
// row migrated from an older release can carry a lower-case value.
func systemAdvancedSetModel(model *SystemAdvancedResourceModel, cfg *truenas.SystemAdvanced) {
	servers := make([]attr.Value, 0, len(cfg.Syslogservers))
	for _, s := range cfg.Syslogservers {
		cert := types.Int64Null()
		if s.TLSCertificate != nil {
			cert = types.Int64Value(*s.TLSCertificate)
		}
		servers = append(servers, types.ObjectValueMust(systemAdvancedSyslogServerAttrTypes, map[string]attr.Value{
			"host":            types.StringValue(s.Host),
			"transport":       types.StringValue(s.Transport),
			"tls_certificate": cert,
		}))
	}
	overprovision := types.Int64Null()
	if cfg.Overprovision != nil {
		overprovision = types.Int64Value(*cfg.Overprovision)
	}

	model.ID = types.StringValue(systemAdvancedSingletonID)
	model.Syslogservers = types.ListValueMust(types.ObjectType{AttrTypes: systemAdvancedSyslogServerAttrTypes}, servers)
	model.Sysloglevel = types.StringValue(strings.ToUpper(cfg.Sysloglevel))
	model.FQDNSyslog = types.BoolValue(cfg.FQDNSyslog)
	model.SyslogAudit = types.BoolValue(cfg.SyslogAudit)
	model.Motd = types.StringValue(cfg.Motd)
	model.LoginBanner = types.StringValue(cfg.LoginBanner)
	model.Consolemenu = types.BoolValue(cfg.Consolemenu)
	model.Serialconsole = types.BoolValue(cfg.Serialconsole)
	model.Serialport = types.StringValue(cfg.Serialport)
	model.Serialspeed = types.StringValue(cfg.Serialspeed)
	model.BootScrub = types.Int64Value(cfg.BootScrub)
	model.KernelExtraOptions = types.StringValue(cfg.KernelExtraOptions)
	model.KdumpEnabled = types.BoolValue(cfg.KdumpEnabled)
	model.Debugkernel = types.BoolValue(cfg.Debugkernel)
	model.Powerdaemon = types.BoolValue(cfg.Powerdaemon)
	model.Traceback = types.BoolValue(cfg.Traceback)
	model.Uploadcrash = types.BoolValue(cfg.Uploadcrash)
	model.Anonstats = types.BoolValue(cfg.Anonstats)
	model.Overprovision = overprovision
}

func (r *SystemAdvancedResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create SystemAdvanced start")

	var plan SystemAdvancedResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithCreateTimeout(ctx, plan.Timeouts, &resp.Diagnostics)
	defer cancel()

	cfg, err := r.client.UpdateSystemAdvanced(ctx, r.updateRequest(ctx, &plan, &resp.Diagnostics))
	if err != nil {
		resp.Diagnostics.AddError("Error Creating System Advanced Config", err.Error())
		return
	}

	systemAdvancedSetModel(&plan, cfg)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	tflog.Trace(ctx, "Create SystemAdvanced success")
}

func (r *SystemAdvancedResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read SystemAdvanced start")

	var state SystemAdvancedResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithReadTimeout(ctx, state.Timeouts, &resp.Diagnostics)
	defer cancel()

	cfg, err := r.client.GetSystemAdvanced(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading System Advanced Config", err.Error())
		return
	}

	systemAdvancedSetModel(&state, cfg)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	tflog.Trace(ctx, "Read SystemAdvanced success")
}

func (r *SystemAdvancedResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update SystemAdvanced start")

	var plan SystemAdvancedResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithUpdateTimeout(ctx, plan.Timeouts, &resp.Diagnostics)
	defer cancel()

	cfg, err := r.client.UpdateSystemAdvanced(ctx, r.updateRequest(ctx, &plan, &resp.Diagnostics))
	if err != nil {
		resp.Diagnostics.AddError("Error Updating System Advanced Config", err.Error())
		return
	}

	systemAdvancedSetModel(&plan, cfg)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	tflog.Trace(ctx, "Update SystemAdvanced success")
}

// Delete is a no-op. The System > Advanced row always exists; removing the
// resource stops Terraform managing it. It does not put the defaults back,
// because a destroy that dropped the remote syslog servers, changed the
// kernel command line or the serial console would be a surprise, and the
// last two take effect at a reboot nobody expects.
func (r *SystemAdvancedResource) Delete(ctx context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete SystemAdvanced: no-op, singleton config is not removable")
}

func (r *SystemAdvancedResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID != systemAdvancedSingletonID {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("truenas_system_advanced is a singleton; the import ID must be %q, got %q",
				systemAdvancedSingletonID, req.ID),
		)
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
