package gaia

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"google.golang.org/api/compute/v1"
)

func resourceGaiaArchetypalCompute() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "Sample resource in the Terraform provider scaffolding.",

		Create: resourceGaiaArchetypalComputeCreate,
		Read:   resourceGaiaArchetypalComputeRead,
		Update: resourceGaiaArchetypalComputeUpdate,
		Delete: resourceGaiaArchetypalComputeDelete,
		/*
			Importer: &schema.ResourceImporter{
				State: resourceComputeInstanceImportState,
			},
		*/
		SchemaVersion: 6,
		//MigrateState:  resourceComputeInstanceMigrateState,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(20 * time.Minute),
			Update: schema.DefaultTimeout(20 * time.Minute),
			Delete: schema.DefaultTimeout(20 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"sample_attribute": {
				// This description is used by the documentation generator and the language server.
				Description: "Sample attribute.",
				Type:        schema.TypeString,
				Optional:    true,
			},
		},
	}
}

func getInstance(config *Config, d *schema.ResourceData) (*compute.Instance, error) {
	project, err := getProject(d, config)
	if err != nil {
		return nil, err
	}
	zone, err := getZone(d, config)
	if err != nil {
		return nil, err
	}
	userAgent, err := generateUserAgentString(d, config.userAgent)
	if err != nil {
		return nil, err
	}
	instance, err := config.NewComputeClient(userAgent).Instances.Get(project, zone, d.Get("name").(string)).Do()
	if err != nil {
		return nil, handleNotFoundError(err, d, fmt.Sprintf("Instance %s", d.Get("name").(string)))
	}
	return instance, nil
}

func getDisk(diskUri string, d *schema.ResourceData, config *Config) (*compute.Disk, error) {
	source, err := ParseDiskFieldValue(diskUri, d, config)
	if err != nil {
		return nil, err
	}

	userAgent, err := generateUserAgentString(d, config.userAgent)
	if err != nil {
		return nil, err
	}

	disk, err := config.NewComputeClient(userAgent).Disks.Get(source.Project, source.Zone, source.Name).Do()
	if err != nil {
		return nil, err
	}

	return disk, err
}

func expandComputeInstance(project string, d *schema.ResourceData, config *Config) (*compute.Instance, error) {
	// Get the machine type
	var machineTypeUrl string
	if mt, ok := d.GetOk("machine_type"); ok {
		machineType, err := ParseMachineTypesFieldValue(mt.(string), d, config)
		if err != nil {
			return nil, fmt.Errorf(
				"Error loading machine type: %s",
				err)
		}
		machineTypeUrl = machineType.RelativeLink()
	}

	// Build up the list of disks

	disks := []*compute.AttachedDisk{}
	if _, hasBootDisk := d.GetOk("boot_disk"); hasBootDisk {
		bootDisk, err := expandBootDisk(d, config, project)
		if err != nil {
			return nil, err
		}
		disks = append(disks, bootDisk)
	}

	if _, hasScratchDisk := d.GetOk("scratch_disk"); hasScratchDisk {
		scratchDisks, err := expandScratchDisks(d, config, project)
		if err != nil {
			return nil, err
		}
		disks = append(disks, scratchDisks...)
	}

	attachedDisksCount := d.Get("attached_disk.#").(int)

	for i := 0; i < attachedDisksCount; i++ {
		diskConfig := d.Get(fmt.Sprintf("attached_disk.%d", i)).(map[string]interface{})
		disk, err := expandAttachedDisk(diskConfig, d, config)
		if err != nil {
			return nil, err
		}

		disks = append(disks, disk)
	}

	scheduling, err := expandScheduling(d.Get("scheduling"))
	if err != nil {
		return nil, fmt.Errorf("Error creating scheduling: %s", err)
	}

	metadata, err := resourceInstanceMetadata(d)
	if err != nil {
		return nil, fmt.Errorf("Error creating metadata: %s", err)
	}

	networkInterfaces, err := expandNetworkInterfaces(d, config)
	if err != nil {
		return nil, fmt.Errorf("Error creating network interfaces: %s", err)
	}
	accels, err := expandInstanceGuestAccelerators(d, config)
	if err != nil {
		return nil, fmt.Errorf("Error creating guest accelerators: %s", err)
	}

	reservationAffinity, err := expandReservationAffinity(d)
	if err != nil {
		return nil, fmt.Errorf("Error creating reservation affinity: %s", err)
	}

	// Create the instance information
	return &compute.Instance{
		CanIpForward:               d.Get("can_ip_forward").(bool),
		Description:                d.Get("description").(string),
		Disks:                      disks,
		MachineType:                machineTypeUrl,
		Metadata:                   metadata,
		Name:                       d.Get("name").(string),
		NetworkInterfaces:          networkInterfaces,
		Tags:                       resourceInstanceTags(d),
		Labels:                     expandLabels(d),
		ServiceAccounts:            expandServiceAccounts(d.Get("service_account").([]interface{})),
		GuestAccelerators:          accels,
		MinCpuPlatform:             d.Get("min_cpu_platform").(string),
		Scheduling:                 scheduling,
		DeletionProtection:         d.Get("deletion_protection").(bool),
		Hostname:                   d.Get("hostname").(string),
		ForceSendFields:            []string{"CanIpForward", "DeletionProtection"},
		ConfidentialInstanceConfig: expandConfidentialInstanceConfig(d),
		AdvancedMachineFeatures:    expandAdvancedMachineFeatures(d),
		ShieldedInstanceConfig:     expandShieldedVmConfigs(d),
		DisplayDevice:              expandDisplayDevice(d),
		ResourcePolicies:           convertStringArr(d.Get("resource_policies").([]interface{})),
		ReservationAffinity:        reservationAffinity,
	}, nil
}

var computeInstanceStatus = []string{
	"PROVISIONING",
	"REPAIRING",
	"RUNNING",
	"STAGING",
	"STOPPED",
	"STOPPING",
	"SUSPENDED",
	"SUSPENDING",
	"TERMINATED",
}

// return all possible Compute instances status except the one passed as parameter
func getAllStatusBut(status string) []string {
	for i, s := range computeInstanceStatus {
		if status == s {
			return append(computeInstanceStatus[:i], computeInstanceStatus[i+1:]...)
		}
	}
	return computeInstanceStatus
}

func waitUntilInstanceHasDesiredStatus(config *Config, d *schema.ResourceData) error {
	desiredStatus := d.Get("desired_status").(string)

	if desiredStatus != "" {
		stateRefreshFunc := func() (interface{}, string, error) {
			instance, err := getInstance(config, d)
			if err != nil || instance == nil {
				log.Printf("Error on InstanceStateRefresh: %s", err)
				return nil, "", err
			}
			return instance.Id, instance.Status, nil
		}
		stateChangeConf := resource.StateChangeConf{
			Delay:      5 * time.Second,
			Pending:    getAllStatusBut(desiredStatus),
			Refresh:    stateRefreshFunc,
			Target:     []string{desiredStatus},
			Timeout:    d.Timeout(schema.TimeoutUpdate),
			MinTimeout: 2 * time.Second,
		}
		_, err := stateChangeConf.WaitForState()

		if err != nil {
			return fmt.Errorf(
				"Error waiting for instance to reach desired status %s: %s", desiredStatus, err)
		}
	}

	return nil
}

func resourceGaiaArchetypalComputeCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	userAgent, err := generateUserAgentString(d, config.userAgent)
	if err != nil {
		return err
	}

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	// Get the zone
	z, err := getZone(d, config)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] Loading zone: %s", z)
	zone, err := config.NewComputeClient(userAgent).Zones.Get(
		project, z).Do()
	if err != nil {
		return fmt.Errorf("Error loading zone '%s': %s", z, err)
	}

	instance, err := expandComputeInstance(project, d, config)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Requesting instance creation")
	op, err := config.NewComputeClient(userAgent).Instances.Insert(project, zone.Name, instance).Do()
	if err != nil {
		return fmt.Errorf("Error creating instance: %s", err)
	}

	// Store the ID now
	d.SetId(fmt.Sprintf("projects/%s/zones/%s/instances/%s", project, z, instance.Name))

	// Wait for the operation to complete
	waitErr := computeOperationWaitTime(config, op, project, "instance to create", userAgent, d.Timeout(schema.TimeoutCreate))
	if waitErr != nil {
		// The resource didn't actually create
		d.SetId("")
		return waitErr
	}

	err = waitUntilInstanceHasDesiredStatus(config, d)
	if err != nil {
		return fmt.Errorf("Error waiting for status: %s", err)
	}

	return resourceComputeInstanceRead(d, meta)
}

func resourceGaiaArchetypalComputeRead(d *schema.ResourceData, meta interface{}) error {
	// use the meta value to retrieve your client from the provider configure method
	// client := meta.(*apiClient)

	return nil
}

func resourceGaiaArchetypalComputeUpdate(d *schema.ResourceData, meta interface{}) error {
	// use the meta value to retrieve your client from the provider configure method
	// client := meta.(*apiClient)

	return nil
}

func resourceGaiaArchetypalComputeDelete(d *schema.ResourceData, meta interface{}) error {
	// use the meta value to retrieve your client from the provider configure method
	// client := meta.(*apiClient)

	return nil
}

func startInstanceOperation(d *schema.ResourceData, config *Config) (*compute.Operation, error) {
	project, err := getProject(d, config)
	if err != nil {
		return nil, err
	}

	zone, err := getZone(d, config)
	if err != nil {
		return nil, err
	}

	userAgent, err := generateUserAgentString(d, config.userAgent)
	if err != nil {
		return nil, err
	}

	// Use beta api directly in order to read network_interface.fingerprint without having to put it in the schema.
	// Change back to getInstance(config, d) once updating alias ips is GA.
	instance, err := config.NewComputeClient(userAgent).Instances.Get(project, zone, d.Get("name").(string)).Do()
	if err != nil {
		return nil, handleNotFoundError(err, d, fmt.Sprintf("Instance %s", instance.Name))
	}

	// Retrieve instance from config to pull encryption keys if necessary
	instanceFromConfig, err := expandComputeInstance(project, d, config)
	if err != nil {
		return nil, err
	}

	var encrypted []*compute.CustomerEncryptionKeyProtectedDisk
	for _, disk := range instanceFromConfig.Disks {
		if disk.DiskEncryptionKey != nil {
			key := compute.CustomerEncryptionKey{RawKey: disk.DiskEncryptionKey.RawKey, KmsKeyName: disk.DiskEncryptionKey.KmsKeyName}
			eDisk := compute.CustomerEncryptionKeyProtectedDisk{Source: disk.Source, DiskEncryptionKey: &key}
			encrypted = append(encrypted, &eDisk)
		}
	}

	var op *compute.Operation

	if len(encrypted) > 0 {
		request := compute.InstancesStartWithEncryptionKeyRequest{Disks: encrypted}
		op, err = config.NewComputeClient(userAgent).Instances.StartWithEncryptionKey(project, zone, instance.Name, &request).Do()
	} else {
		op, err = config.NewComputeClient(userAgent).Instances.Start(project, zone, instance.Name).Do()
	}

	return op, err
}

func expandAttachedDisk(diskConfig map[string]interface{}, d *schema.ResourceData, meta interface{}) (*compute.AttachedDisk, error) {
	config := meta.(*Config)

	s := diskConfig["source"].(string)
	var sourceLink string
	if strings.Contains(s, "regions/") {
		source, err := ParseRegionDiskFieldValue(s, d, config)
		if err != nil {
			return nil, err
		}
		sourceLink = source.RelativeLink()
	} else {
		source, err := ParseDiskFieldValue(s, d, config)
		if err != nil {
			return nil, err
		}
		sourceLink = source.RelativeLink()
	}

	disk := &compute.AttachedDisk{
		Source: sourceLink,
	}

	if v, ok := diskConfig["mode"]; ok {
		disk.Mode = v.(string)
	}

	if v, ok := diskConfig["device_name"]; ok {
		disk.DeviceName = v.(string)
	}

	keyValue, keyOk := diskConfig["disk_encryption_key_raw"]
	if keyOk {
		if keyValue != "" {
			disk.DiskEncryptionKey = &compute.CustomerEncryptionKey{
				RawKey: keyValue.(string),
			}
		}
	}

	kmsValue, kmsOk := diskConfig["kms_key_self_link"]
	if kmsOk {
		if keyOk && keyValue != "" && kmsValue != "" {
			return nil, errors.New("Only one of kms_key_self_link and disk_encryption_key_raw can be set")
		}
		if kmsValue != "" {
			disk.DiskEncryptionKey = &compute.CustomerEncryptionKey{
				KmsKeyName: kmsValue.(string),
			}
		}
	}
	return disk, nil
}

// See comment on expandInstanceTemplateGuestAccelerators regarding why this
// code is duplicated.
func expandInstanceGuestAccelerators(d TerraformResourceData, config *Config) ([]*compute.AcceleratorConfig, error) {
	configs, ok := d.GetOk("guest_accelerator")
	if !ok {
		return nil, nil
	}
	accels := configs.([]interface{})
	guestAccelerators := make([]*compute.AcceleratorConfig, 0, len(accels))
	for _, raw := range accels {
		data := raw.(map[string]interface{})
		if data["count"].(int) == 0 {
			continue
		}
		at, err := ParseAcceleratorFieldValue(data["type"].(string), d, config)
		if err != nil {
			return nil, fmt.Errorf("cannot parse accelerator type: %v", err)
		}
		guestAccelerators = append(guestAccelerators, &compute.AcceleratorConfig{
			AcceleratorCount: int64(data["count"].(int)),
			AcceleratorType:  at.RelativeLink(),
		})
	}

	return guestAccelerators, nil
}

// suppressEmptyGuestAcceleratorDiff is used to work around perpetual diff
// issues when a count of `0` guest accelerators is desired. This may occur when
// guest_accelerator support is controlled via a module variable. E.g.:
//
// 		guest_accelerators {
//      	count = "${var.enable_gpu ? var.gpu_count : 0}"
//          ...
// 		}
// After reconciling the desired and actual state, we would otherwise see a
// perpetual resembling:
// 		[] != [{"count":0, "type": "nvidia-tesla-k80"}]
func suppressEmptyGuestAcceleratorDiff(_ context.Context, d *schema.ResourceDiff, meta interface{}) error {
	oldi, newi := d.GetChange("guest_accelerator")

	old, ok := oldi.([]interface{})
	if !ok {
		return fmt.Errorf("Expected old guest accelerator diff to be a slice")
	}

	new, ok := newi.([]interface{})
	if !ok {
		return fmt.Errorf("Expected new guest accelerator diff to be a slice")
	}

	if len(old) != 0 && len(new) != 1 {
		return nil
	}

	firstAccel, ok := new[0].(map[string]interface{})
	if !ok {
		return fmt.Errorf("Unable to type assert guest accelerator")
	}

	if firstAccel["count"].(int) == 0 {
		if err := d.Clear("guest_accelerator"); err != nil {
			return err
		}
	}

	return nil
}

// return an error if the desired_status field is set to a value other than RUNNING on Create.
func desiredStatusDiff(_ context.Context, diff *schema.ResourceDiff, meta interface{}) error {
	// when creating an instance, name is not set
	oldName, _ := diff.GetChange("name")

	if oldName == nil || oldName == "" {
		_, newDesiredStatus := diff.GetChange("desired_status")

		if newDesiredStatus == nil || newDesiredStatus == "" {
			return nil
		} else if newDesiredStatus != "RUNNING" {
			return fmt.Errorf("When creating an instance, desired_status can only accept RUNNING value")
		}
		return nil
	}

	return nil
}

func resourceComputeInstanceDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	userAgent, err := generateUserAgentString(d, config.userAgent)
	if err != nil {
		return err
	}

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	zone, err := getZone(d, config)
	if err != nil {
		return err
	}
	log.Printf("[INFO] Requesting instance deletion: %s", d.Get("name").(string))

	if d.Get("deletion_protection").(bool) {
		return fmt.Errorf("Cannot delete instance %s: instance Deletion Protection is enabled. Set deletion_protection to false for this resource and run \"terraform apply\" before attempting to delete it.", d.Get("name").(string))
	} else {
		op, err := config.NewComputeClient(userAgent).Instances.Delete(project, zone, d.Get("name").(string)).Do()
		if err != nil {
			return fmt.Errorf("Error deleting instance: %s", err)
		}

		// Wait for the operation to complete
		opErr := computeOperationWaitTime(config, op, project, "instance to delete", userAgent, d.Timeout(schema.TimeoutDelete))
		if opErr != nil {
			// Refresh operation to check status
			op, _ = config.NewComputeClient(userAgent).ZoneOperations.Get(project, zone, strconv.FormatUint(op.Id, 10)).Do()
			// Do not return an error if the operation actually completed
			if op == nil || op.Status != "DONE" {
				return opErr
			}
		}

		d.SetId("")
		return nil
	}
}

func resourceComputeInstanceImportState(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	config := meta.(*Config)
	if err := parseImportId([]string{
		"projects/(?P<project>[^/]+)/zones/(?P<zone>[^/]+)/instances/(?P<name>[^/]+)",
		"(?P<project>[^/]+)/(?P<zone>[^/]+)/(?P<name>[^/]+)",
		"(?P<name>[^/]+)",
	}, d, config); err != nil {
		return nil, err
	}

	// Replace import id for the resource id
	id, err := replaceVars(d, config, "projects/{{project}}/zones/{{zone}}/instances/{{name}}")
	if err != nil {
		return nil, fmt.Errorf("Error constructing id: %s", err)
	}
	d.SetId(id)

	return []*schema.ResourceData{d}, nil
}

func expandBootDisk(d *schema.ResourceData, config *Config, project string) (*compute.AttachedDisk, error) {
	userAgent, err := generateUserAgentString(d, config.userAgent)
	if err != nil {
		return nil, err
	}

	disk := &compute.AttachedDisk{
		AutoDelete: d.Get("boot_disk.0.auto_delete").(bool),
		Boot:       true,
	}

	if v, ok := d.GetOk("boot_disk.0.device_name"); ok {
		disk.DeviceName = v.(string)
	}

	if v, ok := d.GetOk("boot_disk.0.disk_encryption_key_raw"); ok {
		if v != "" {
			disk.DiskEncryptionKey = &compute.CustomerEncryptionKey{
				RawKey: v.(string),
			}
		}
	}

	if v, ok := d.GetOk("boot_disk.0.kms_key_self_link"); ok {
		if v != "" {
			disk.DiskEncryptionKey = &compute.CustomerEncryptionKey{
				KmsKeyName: v.(string),
			}
		}
	}

	if v, ok := d.GetOk("boot_disk.0.source"); ok {
		source, err := ParseDiskFieldValue(v.(string), d, config)
		if err != nil {
			return nil, err
		}
		disk.Source = source.RelativeLink()
	}

	if _, ok := d.GetOk("boot_disk.0.initialize_params"); ok {
		disk.InitializeParams = &compute.AttachedDiskInitializeParams{}

		if v, ok := d.GetOk("boot_disk.0.initialize_params.0.size"); ok {
			disk.InitializeParams.DiskSizeGb = int64(v.(int))
		}

		if v, ok := d.GetOk("boot_disk.0.initialize_params.0.type"); ok {
			diskTypeName := v.(string)
			diskType, err := readDiskType(config, d, diskTypeName)
			if err != nil {
				return nil, fmt.Errorf("Error loading disk type '%s': %s", diskTypeName, err)
			}
			disk.InitializeParams.DiskType = diskType.RelativeLink()
		}

		if v, ok := d.GetOk("boot_disk.0.initialize_params.0.image"); ok {
			imageName := v.(string)
			imageUrl, err := resolveImage(config, project, imageName, userAgent)
			if err != nil {
				return nil, fmt.Errorf("Error resolving image name '%s': %s", imageName, err)
			}

			disk.InitializeParams.SourceImage = imageUrl
		}

		if _, ok := d.GetOk("boot_disk.0.initialize_params.0.labels"); ok {
			disk.InitializeParams.Labels = expandStringMap(d, "boot_disk.0.initialize_params.0.labels")
		}
	}

	if v, ok := d.GetOk("boot_disk.0.mode"); ok {
		disk.Mode = v.(string)
	}

	return disk, nil
}

func flattenBootDisk(d *schema.ResourceData, disk *compute.AttachedDisk, config *Config) []map[string]interface{} {
	result := map[string]interface{}{
		"auto_delete": disk.AutoDelete,
		"device_name": disk.DeviceName,
		"mode":        disk.Mode,
		"source":      ConvertSelfLinkToV1(disk.Source),
		// disk_encryption_key_raw is not returned from the API, so copy it from what the user
		// originally specified to avoid diffs.
		"disk_encryption_key_raw": d.Get("boot_disk.0.disk_encryption_key_raw"),
	}

	diskDetails, err := getDisk(disk.Source, d, config)
	if err != nil {
		log.Printf("[WARN] Cannot retrieve boot disk details: %s", err)

		if _, ok := d.GetOk("boot_disk.0.initialize_params.#"); ok {
			// If we can't read the disk details due to permission for instance,
			// copy the initialize_params from what the user originally specified to avoid diffs.
			m := d.Get("boot_disk.0.initialize_params")
			result["initialize_params"] = m
		}
	} else {
		result["initialize_params"] = []map[string]interface{}{{
			"type": GetResourceNameFromSelfLink(diskDetails.Type),
			// If the config specifies a family name that doesn't match the image name, then
			// the diff won't be properly suppressed. See DiffSuppressFunc for this field.
			"image":  diskDetails.SourceImage,
			"size":   diskDetails.SizeGb,
			"labels": diskDetails.Labels,
		}}
	}

	if disk.DiskEncryptionKey != nil {
		if disk.DiskEncryptionKey.Sha256 != "" {
			result["disk_encryption_key_sha256"] = disk.DiskEncryptionKey.Sha256
		}
		if disk.DiskEncryptionKey.KmsKeyName != "" {
			// The response for crypto keys often includes the version of the key which needs to be removed
			// format: projects/<project>/locations/<region>/keyRings/<keyring>/cryptoKeys/<key>/cryptoKeyVersions/1
			result["kms_key_self_link"] = strings.Split(disk.DiskEncryptionKey.KmsKeyName, "/cryptoKeyVersions")[0]
		}
	}

	return []map[string]interface{}{result}
}

func expandScratchDisks(d *schema.ResourceData, config *Config, project string) ([]*compute.AttachedDisk, error) {
	diskType, err := readDiskType(config, d, "local-ssd")
	if err != nil {
		return nil, fmt.Errorf("Error loading disk type 'local-ssd': %s", err)
	}

	n := d.Get("scratch_disk.#").(int)
	scratchDisks := make([]*compute.AttachedDisk, 0, n)
	for i := 0; i < n; i++ {
		scratchDisks = append(scratchDisks, &compute.AttachedDisk{
			AutoDelete: true,
			Type:       "SCRATCH",
			Interface:  d.Get(fmt.Sprintf("scratch_disk.%d.interface", i)).(string),
			InitializeParams: &compute.AttachedDiskInitializeParams{
				DiskType: diskType.RelativeLink(),
			},
		})
	}

	return scratchDisks, nil
}

func flattenScratchDisk(disk *compute.AttachedDisk) map[string]interface{} {
	result := map[string]interface{}{
		"interface": disk.Interface,
	}
	return result
}

func hash256(raw string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(decoded)
	return base64.StdEncoding.EncodeToString(h[:]), nil
}

func serviceAccountDiffSuppress(k, old, new string, d *schema.ResourceData) bool {
	if k != "service_account.#" {
		return false
	}

	o, n := d.GetChange("service_account")
	var l []interface{}
	if old == "0" && new == "1" {
		l = n.([]interface{})
	} else if new == "0" && old == "1" {
		l = o.([]interface{})
	} else {
		// we don't have one set and one unset, so don't suppress the diff
		return false
	}

	// suppress changes between { } and {scopes:[]}
	if l[0] != nil {
		contents := l[0].(map[string]interface{})
		if scopes, ok := contents["scopes"]; ok {
			a := scopes.(*schema.Set).List()
			if a != nil && len(a) > 0 {
				return false
			}
		}
	}
	return true
}
