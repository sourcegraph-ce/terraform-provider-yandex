package yandex

import (
	"fmt"
	log "github.com/sourcegraph-ce/logrus"
	"sort"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/hashicorp/terraform-plugin-sdk/helper/hashcode"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"google.golang.org/genproto/googleapis/type/dayofweek"
	"google.golang.org/genproto/googleapis/type/timeofday"
	"google.golang.org/genproto/protobuf/field_mask"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/k8s/v1"
)

const (
	yandexKubernetesClusterCreateTimeout  = 30 * time.Minute
	yandexKubernetesClusterReadTimeout    = 5 * time.Minute
	yandexKubernetesClusterDefaultTimeout = 20 * time.Minute
	yandexKubernetesClusterUpdateTimeout  = 30 * time.Minute
)

func resourceYandexKubernetesCluster() *schema.Resource {
	return &schema.Resource{
		Create: resourceYandexKubernetesClusterCreate,
		Read:   resourceYandexKubernetesClusterRead,
		Update: resourceYandexKubernetesClusterUpdate,
		Delete: resourceYandexKubernetesClusterDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(yandexKubernetesClusterCreateTimeout),
			Read:   schema.DefaultTimeout(yandexKubernetesClusterReadTimeout),
			Update: schema.DefaultTimeout(yandexKubernetesClusterUpdateTimeout),
			Delete: schema.DefaultTimeout(yandexKubernetesClusterDefaultTimeout),
		},
		Schema: map[string]*schema.Schema{
			"network_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"service_account_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"node_service_account_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"master": {
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"version": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"public_ip": {
							Type:     schema.TypeBool,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},
						"maintenance_policy": {
							Type:     schema.TypeList,
							MaxItems: 1,
							Computed: true,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"auto_upgrade": {
										Type:     schema.TypeBool,
										Required: true,
									},
									"maintenance_window": {
										Type:     schema.TypeSet,
										Optional: true,
										Set:      dayOfWeekHash,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"day": {
													Type:             schema.TypeString,
													Optional:         true,
													Computed:         true,
													ValidateFunc:     validateParsableValue(parseDayOfWeek),
													DiffSuppressFunc: shouldSuppressDiffForDayOfWeek,
												},
												"start_time": {
													Type:             schema.TypeString,
													Required:         true,
													ValidateFunc:     validateParsableValue(parseDayTime),
													DiffSuppressFunc: shouldSuppressDiffForTimeOfDay,
												},
												"duration": {
													Type:             schema.TypeString,
													Required:         true,
													ValidateFunc:     validateParsableValue(parseDuration),
													DiffSuppressFunc: shouldSuppressDiffForTimeDuration,
												},
											},
										},
									},
								},
							},
						},
						"zonal": {
							Type:          schema.TypeList,
							Computed:      true,
							Optional:      true,
							ForceNew:      true,
							ConflictsWith: []string{"master.0.regional"},
							MaxItems:      1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"zone": {
										Type:     schema.TypeString,
										Optional: true,
										Computed: true,
										ForceNew: true,
									},
									"subnet_id": {
										Type:     schema.TypeString,
										Optional: true,
										ForceNew: true,
									},
								},
							},
						},
						"regional": {
							Type:          schema.TypeList,
							MaxItems:      1,
							Optional:      true,
							Computed:      true,
							ForceNew:      true,
							ConflictsWith: []string{"master.0.zonal"},
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"region": {
										Type:     schema.TypeString,
										Required: true,
										ForceNew: true,
									},
									"location": {
										Type:     schema.TypeList,
										Optional: true,
										Computed: true,
										ForceNew: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"zone": {
													Type:     schema.TypeString,
													Optional: true,
													ForceNew: true,
												},
												"subnet_id": {
													Type:     schema.TypeString,
													Optional: true,
													ForceNew: true,
												},
											},
										},
									},
								},
							},
						},
						"internal_v4_address": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"external_v4_address": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"internal_v4_endpoint": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"external_v4_endpoint": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"cluster_ca_certificate": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"version_info": {
							Type:     schema.TypeList,
							Computed: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"current_version": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"new_revision_available": {
										Type:     schema.TypeBool,
										Computed: true,
									},
									"new_revision_summary": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"version_deprecated": {
										Type:     schema.TypeBool,
										Computed: true,
									},
								},
							},
						},
					},
				},
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"folder_id": {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				ForceNew: true,
			},
			"description": {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
			},
			"labels": {
				Type:     schema.TypeMap,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
				Optional: true,
				Computed: true,
			},
			"cluster_ipv4_range": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"node_ipv4_cidr_mask_size": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
				Default:  24,
			},
			"service_ipv4_range": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"release_channel": {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				ForceNew: true,
			},
			"health": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"created_at": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"network_policy_provider": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"kms_provider": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key_id": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},
		},
	}
}

func resourceYandexKubernetesClusterCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	req, err := prepareCreateKubernetesClusterRequest(d, config)
	if err != nil {
		return err
	}

	ctx, cancel := config.ContextWithTimeout(d.Timeout(schema.TimeoutCreate))
	defer cancel()

	op, err := config.sdk.WrapOperation(config.sdk.Kubernetes().Cluster().Create(ctx, req))
	if err != nil {
		return fmt.Errorf("error while requesting API to create Kubernetes cluster: %s", err)
	}

	protoMetadata, err := op.Metadata()
	if err != nil {
		return fmt.Errorf("error while get Kubernetes cluster create operation metadata: %s", err)
	}

	md, ok := protoMetadata.(*k8s.CreateClusterMetadata)
	if !ok {
		return fmt.Errorf("could not get Kubernetes cluster ID from create operation metadata")
	}

	d.SetId(md.GetClusterId())

	err = op.Wait(ctx)
	if err != nil {
		return fmt.Errorf("error while waiting operation to create Kubernetes cluster: %s", err)
	}

	if _, err := op.Response(); err != nil {
		return fmt.Errorf("Kubernetes cluster creation failed: %s", err)
	}

	return resourceYandexKubernetesClusterRead(d, meta)
}

func resourceYandexKubernetesClusterRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	clusterID := d.Id()

	ctx, cancel := config.ContextWithTimeout(d.Timeout(schema.TimeoutRead))
	defer cancel()

	cluster, err := config.sdk.Kubernetes().Cluster().Get(ctx, &k8s.GetClusterRequest{
		ClusterId: clusterID,
	})

	if err != nil {
		return handleNotFoundError(err, d, fmt.Sprintf("Kubernetes cluster with ID %q", clusterID))
	}

	return flattenKubernetesClusterAttributes(cluster, d, true)
}

var updateKubernetesClusterFieldsMap = map[string]string{
	"name":                        "name",
	"description":                 "description",
	"labels":                      "labels",
	"service_account_id":          "service_account_id",
	"node_service_account_id":     "node_service_account_id",
	"master.0.version":            "master_spec.version.version",
	"master.0.maintenance_policy": "master_spec.maintenance_policy",
}

func resourceYandexKubernetesClusterUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	clusterID := d.Id()
	log.Printf("[DEBUG] updating Kubernetes cluster %q", clusterID)

	req, err := getKubernetesClusterUpdateRequest(d)
	if err != nil {
		return err
	}

	var updatePath []string
	for field, path := range updateKubernetesClusterFieldsMap {
		if d.HasChange(field) {
			updatePath = append(updatePath, path)
		}
	}

	if len(updatePath) == 0 {
		return fmt.Errorf("error while updating Kubernetes cluster, didn't detect any changes")
	}

	req.UpdateMask = &field_mask.FieldMask{Paths: updatePath}
	ctx, cancel := config.ContextWithTimeout(d.Timeout(schema.TimeoutUpdate))
	defer cancel()

	op, err := config.sdk.WrapOperation(config.sdk.Kubernetes().Cluster().Update(ctx, req))
	if err != nil {
		return fmt.Errorf("error while requesting API to update Kubernetes cluster %q: %s", clusterID, err)
	}

	err = op.Wait(ctx)
	if err != nil {
		return fmt.Errorf("error updating Kubernetes cluster %q: %s", clusterID, err)
	}

	return resourceYandexKubernetesClusterRead(d, meta)
}

func getKubernetesClusterUpdateRequest(d *schema.ResourceData) (*k8s.UpdateClusterRequest, error) {
	labels, err := expandLabels(d.Get("labels"))
	if err != nil {
		return nil, fmt.Errorf("error expanding labels while updating Kubernetes cluster: %s", err)
	}

	mp, err := getKubernetesClusterMasterMaintenancePolicy(d)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster master maintenance policy: %s", err)
	}

	req := &k8s.UpdateClusterRequest{
		ClusterId:            d.Id(),
		Name:                 d.Get("name").(string),
		Description:          d.Get("description").(string),
		Labels:               labels,
		ServiceAccountId:     d.Get("service_account_id").(string),
		NodeServiceAccountId: d.Get("node_service_account_id").(string),
		MasterSpec: &k8s.MasterUpdateSpec{
			Version: &k8s.UpdateVersionSpec{
				Specifier: &k8s.UpdateVersionSpec_Version{
					Version: d.Get("master.0.version").(string),
				},
			},
			MaintenancePolicy: mp,
		},
	}

	return req, nil

}

func resourceYandexKubernetesClusterDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	clusterID := d.Id()

	log.Printf("[DEBUG] Deleting Kubernetes cluster %q", d.Id())

	req := &k8s.DeleteClusterRequest{
		ClusterId: clusterID,
	}

	ctx, cancel := config.ContextWithTimeout(d.Timeout(schema.TimeoutDelete))
	defer cancel()

	op, err := config.sdk.WrapOperation(config.sdk.Kubernetes().Cluster().Delete(ctx, req))
	if err != nil {
		return handleNotFoundError(err, d, fmt.Sprintf("Kubernetes cluster %q", clusterID))
	}

	err = op.Wait(ctx)
	if err != nil {
		return err
	}

	_, err = op.Response()
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Finished deleting Kubernetes cluster %q", d.Id())
	return nil
}

func prepareCreateKubernetesClusterRequest(d *schema.ResourceData, meta *Config) (*k8s.CreateClusterRequest, error) {
	folderID, err := getFolderID(d, meta)
	if err != nil {
		return nil, fmt.Errorf("error getting folder ID while creating Kubernetes cluster: %s", err)
	}

	labels, err := expandLabels(d.Get("labels"))
	if err != nil {
		return nil, fmt.Errorf("error expanding labels while creating Kubernetes cluster: %s", err)
	}

	masterSpec, err := getKubernetesClusterMasterSpec(d, meta)
	if err != nil {
		return nil, fmt.Errorf("error getting master spec while creating Kubernetes cluster: %s", err)
	}
	releaseChannel, err := getKubernetesClusterReleaseChannel(d)
	if err != nil {
		return nil, err
	}
	networkPolicy, err := getKubernetesClusterNetworkPolicy(d)
	if err != nil {
		return nil, err
	}

	req := &k8s.CreateClusterRequest{
		FolderId:             folderID,
		Name:                 d.Get("name").(string),
		Description:          d.Get("description").(string),
		Labels:               labels,
		NetworkId:            d.Get("network_id").(string),
		MasterSpec:           masterSpec,
		IpAllocationPolicy:   getIPAllocationPolicy(d),
		ServiceAccountId:     d.Get("service_account_id").(string),
		NodeServiceAccountId: d.Get("node_service_account_id").(string),
		ReleaseChannel:       releaseChannel,
		NetworkPolicy:        networkPolicy,
		KmsProvider:          getKubernetesClusterKMSProvider(d),
	}

	return req, nil
}

func getIPAllocationPolicy(d *schema.ResourceData) *k8s.IPAllocationPolicy {
	p := &k8s.IPAllocationPolicy{
		ClusterIpv4CidrBlock: d.Get("cluster_ipv4_range").(string),
		NodeIpv4CidrMaskSize: int64(d.Get("node_ipv4_cidr_mask_size").(int)),
		ServiceIpv4CidrBlock: d.Get("service_ipv4_range").(string),
	}

	return p
}

func getKubernetesClusterReleaseChannels() string {
	var values []string
	for k := range k8s.ReleaseChannel_value {
		values = append(values, k)
	}
	sort.Strings(values)

	return strings.Join(values, ",")
}

func getKubernetesClusterReleaseChannel(d *schema.ResourceData) (k8s.ReleaseChannel, error) {
	c, ok := d.GetOk("release_channel")
	if ok {
		if ch, ok := k8s.ReleaseChannel_value[c.(string)]; ok {
			return k8s.ReleaseChannel(ch), nil
		}

		err := fmt.Errorf("invalid release_channel field value, possible values: %v", getKubernetesClusterReleaseChannels())
		return k8s.ReleaseChannel_RELEASE_CHANNEL_UNSPECIFIED, err
	}

	return k8s.ReleaseChannel_RELEASE_CHANNEL_UNSPECIFIED, nil
}

func getKubernetesClusterNetworkPolicyProviders() string {
	var values []string
	for k, v := range k8s.NetworkPolicy_Provider_value {
		if v == int32(k8s.NetworkPolicy_PROVIDER_UNSPECIFIED) {
			continue
		}
		values = append(values, k)
	}
	sort.Strings(values)

	return strings.Join(values, ",")
}

func getKubernetesClusterNetworkPolicy(d *schema.ResourceData) (*k8s.NetworkPolicy, error) {
	provName, ok := d.GetOk("network_policy_provider")
	if !ok {
		return nil, nil
	}
	prov, ok := k8s.NetworkPolicy_Provider_value[strings.ToUpper(provName.(string))]
	if ok && prov != int32(k8s.NetworkPolicy_PROVIDER_UNSPECIFIED) {
		return &k8s.NetworkPolicy{
			Provider: k8s.NetworkPolicy_Provider(prov),
		}, nil
	}
	return nil, fmt.Errorf("invalid network_policy_provider field value, possible values: %v", getKubernetesClusterNetworkPolicyProviders())
}

func getKubernetesClusterKMSProvider(d *schema.ResourceData) *k8s.KMSProvider {
	kmsKeyID, ok := d.Get("kms_provider.0.key_id").(string)
	if !ok {
		return nil
	}
	return &k8s.KMSProvider{
		KeyId: kmsKeyID,
	}
}

func getKubernetesClusterMasterSpec(d *schema.ResourceData, meta *Config) (*k8s.MasterSpec, error) {
	spec := &k8s.MasterSpec{
		Version:    d.Get("master.0.version").(string),
		MasterType: nil,
	}

	var err error
	if spec.MaintenancePolicy, err = getKubernetesClusterMasterMaintenancePolicy(d); err != nil {
		return nil, err
	}

	if _, ok := d.GetOk("master.0.zonal"); ok {
		spec.MasterType = getKubernetesClusterZonalMaster(d, meta)
		return spec, nil
	}

	if _, ok := d.GetOk("master.0.regional"); ok {
		spec.MasterType = getKubernetesClusterRegionalMaster(d, meta)
		return spec, nil
	}

	return nil, fmt.Errorf("either zonal or regional master should be specified for Kubernetes cluster")
}

func getKubernetesClusterMasterMaintenancePolicy(d *schema.ResourceData) (*k8s.MasterMaintenancePolicy, error) {
	if _, ok := d.GetOk("master.0.maintenance_policy"); !ok {
		return nil, nil
	}

	p := &k8s.MasterMaintenancePolicy{
		AutoUpgrade: d.Get("master.0.maintenance_policy.0.auto_upgrade").(bool),
	}

	if mw, ok := d.GetOk("master.0.maintenance_policy.0.maintenance_window"); ok {
		var err error
		if p.MaintenanceWindow, err = expandMaintenanceWindow(mw.(*schema.Set).List()); err != nil {
			return nil, err
		}
	}

	return p, nil
}

func getKubernetesClusterZonalMaster(d *schema.ResourceData, meta *Config) *k8s.MasterSpec_ZonalMasterSpec {
	return &k8s.MasterSpec_ZonalMasterSpec{
		ZonalMasterSpec: &k8s.ZonalMasterSpec{
			ZoneId:                getZonalMasterZone(d, meta),
			InternalV4AddressSpec: getZonalMasterInternalAddressSpec(d),
			ExternalV4AddressSpec: getZonalMasterExternalAddressSpec(d),
		},
	}
}

func getKubernetesClusterRegionalMaster(d *schema.ResourceData, _ *Config) *k8s.MasterSpec_RegionalMasterSpec {
	return &k8s.MasterSpec_RegionalMasterSpec{
		RegionalMasterSpec: &k8s.RegionalMasterSpec{
			RegionId:              d.Get("master.0.regional.0.region").(string),
			Locations:             getKubernetesClusterRegionalMasterLocations(d),
			ExternalV4AddressSpec: getZonalMasterExternalAddressSpec(d),
		},
	}
}

func getKubernetesClusterRegionalMasterLocations(d *schema.ResourceData) []*k8s.MasterLocation {
	var locations []*k8s.MasterLocation
	locationCount := d.Get("master.0.regional.0.location.#").(int)
	for i := 0; i < locationCount; i++ {
		location := d.Get(fmt.Sprintf("master.0.regional.0.location.%d", i)).(map[string]interface{})
		locationSpec := &k8s.MasterLocation{}

		if zone, ok := location["zone"]; ok {
			locationSpec.ZoneId = zone.(string)
		}

		if subnet, ok := location["subnet_id"]; ok {
			locationSpec.InternalV4AddressSpec = &k8s.InternalAddressSpec{
				SubnetId: subnet.(string),
			}
		}

		locations = append(locations, locationSpec)
	}
	return locations
}

func getZonalMasterZone(d *schema.ResourceData, config *Config) string {
	res, ok := d.GetOk("master.0.zonal.0.zone")
	if !ok {
		return config.Zone
	}
	return res.(string)
}

func getZonalMasterInternalAddressSpec(d *schema.ResourceData) *k8s.InternalAddressSpec {
	res, ok := d.GetOk("master.0.zonal.0.subnet_id")
	if ok {
		return &k8s.InternalAddressSpec{
			SubnetId: res.(string),
		}
	}
	return nil
}

func getZonalMasterExternalAddressSpec(d *schema.ResourceData) *k8s.ExternalAddressSpec {
	publicIP, ok := d.GetOk("master.0.public_ip")
	if ok && publicIP.(bool) {
		return &k8s.ExternalAddressSpec{}
	}

	return nil
}

func flattenKubernetesClusterAttributes(cluster *k8s.Cluster, d *schema.ResourceData, clusterResource bool) error {
	createdAt, err := getTimestamp(cluster.CreatedAt)
	if err != nil {
		return err
	}

	d.Set("created_at", createdAt)
	d.Set("folder_id", cluster.FolderId)
	d.Set("name", cluster.Name)
	d.Set("description", cluster.Description)
	d.Set("status", strings.ToLower(cluster.Status.String()))
	d.Set("health", strings.ToLower(cluster.Health.String()))
	d.Set("network_id", cluster.NetworkId)
	d.Set("service_account_id", cluster.ServiceAccountId)
	d.Set("node_service_account_id", cluster.NodeServiceAccountId)
	d.Set("release_channel", cluster.ReleaseChannel.String())
	d.Set("cluster_ipv4_range", cluster.GetIpAllocationPolicy().GetClusterIpv4CidrBlock())
	d.Set("node_ipv4_cidr_mask_size", cluster.GetIpAllocationPolicy().GetNodeIpv4CidrMaskSize())
	d.Set("service_ipv4_range", cluster.GetIpAllocationPolicy().GetServiceIpv4CidrBlock())
	if np := cluster.GetNetworkPolicy(); np != nil {
		if prov := np.GetProvider(); prov != k8s.NetworkPolicy_PROVIDER_UNSPECIFIED {
			d.Set("network_policy_provider", prov.String())
		}
	}
	if kms := cluster.GetKmsProvider(); kms != nil {
		if keyID := kms.GetKeyId(); keyID != "" {
			if err := d.Set("kms_provider", []map[string]interface{}{
				{"key_id": keyID},
			}); err != nil {
				return err
			}
		}
	}

	if err := d.Set("labels", cluster.Labels); err != nil {
		return err
	}

	h, err := flattenKubernetesMaster(cluster)
	if err != nil {
		return err
	}

	if clusterResource {
		h.fillClusterMasterResourceFields(cluster, d)
	} else {
		d.Set("cluster_id", cluster.Id)
	}

	err = d.Set("master", h.schema())
	if err != nil {
		return err
	}

	d.SetId(cluster.Id)
	return nil
}

type masterSchemaHelper struct {
	zonalMaster    map[string]interface{}
	regionalMaster map[string]interface{}
	versionInfo    map[string]interface{}
	master         map[string]interface{}
}

func constructKubernetesMasterSchemaHelper() *masterSchemaHelper {
	helper := &masterSchemaHelper{}
	helper.versionInfo = map[string]interface{}{}
	helper.master = map[string]interface{}{
		"version_info": []map[string]interface{}{
			helper.versionInfo,
		},
	}
	return helper
}

func (h *masterSchemaHelper) schema() []map[string]interface{} {
	return []map[string]interface{}{h.master}
}

func (h *masterSchemaHelper) fillClusterMasterResourceFields(cluster *k8s.Cluster, d *schema.ResourceData) {
	if subnet, ok := d.GetOk("master.0.zonal.0.subnet_id"); ok {
		h.getZonalMaster()["subnet_id"] = subnet
	}

	if region, ok := d.GetOk("master.0.regional.0.region"); ok {
		h.getRegionalMaster()["region"] = region
	}

	if locations, ok := d.GetOk("master.0.regional.0.location"); ok {
		h.getRegionalMaster()["location"] = locations
	}
}

func (h *masterSchemaHelper) getZonalMaster() map[string]interface{} {
	if h.zonalMaster == nil {
		h.zonalMaster = map[string]interface{}{}
		h.master["zonal"] = []map[string]interface{}{
			h.zonalMaster,
		}
	}

	return h.zonalMaster
}

func (h *masterSchemaHelper) getRegionalMaster() map[string]interface{} {
	if h.regionalMaster == nil {
		h.regionalMaster = map[string]interface{}{}
		h.master["regional"] = []map[string]interface{}{
			h.regionalMaster,
		}
	}

	return h.regionalMaster
}

func (h *masterSchemaHelper) flattenMasterMaintenancePolicy(m *k8s.MasterMaintenancePolicy) error {
	maintenanceWindow, err := flattenMaintenanceWindow(m.GetMaintenanceWindow())
	if err != nil {
		return err
	}

	h.master["maintenance_policy"] = []map[string]interface{}{
		{
			"auto_upgrade":       m.GetAutoUpgrade(),
			"maintenance_window": maintenanceWindow,
		},
	}

	return nil
}

func (h *masterSchemaHelper) flattenClusterZonalMaster(m *k8s.Master_ZonalMaster) {
	h.master["internal_v4_address"] = m.ZonalMaster.GetInternalV4Address()
	h.master["external_v4_address"] = m.ZonalMaster.GetExternalV4Address()

	h.getZonalMaster()["zone"] = m.ZonalMaster.GetZoneId()
}

func (h *masterSchemaHelper) flattenClusterRegionalMaster(m *k8s.Master_RegionalMaster) {
	h.master["internal_v4_address"] = m.RegionalMaster.GetInternalV4Address()
	h.master["external_v4_address"] = m.RegionalMaster.GetExternalV4Address()

	h.getRegionalMaster()["region"] = m.RegionalMaster.GetRegionId()
}

func flattenKubernetesMaster(cluster *k8s.Cluster) (*masterSchemaHelper, error) {
	h := constructKubernetesMasterSchemaHelper()
	clusterMaster := cluster.GetMaster()
	if clusterMaster == nil {
		return nil, fmt.Errorf("failed to get cluster master spec")
	}

	h.master["version"] = clusterMaster.GetVersion()
	h.master["public_ip"] = clusterMaster.GetEndpoints().GetExternalV4Endpoint() != ""
	h.master["internal_v4_endpoint"] = clusterMaster.GetEndpoints().GetInternalV4Endpoint()
	h.master["external_v4_endpoint"] = clusterMaster.GetEndpoints().GetExternalV4Endpoint()
	h.master["cluster_ca_certificate"] = clusterMaster.GetMasterAuth().GetClusterCaCertificate()

	p := clusterMaster.GetMaintenancePolicy()
	if p == nil {
		return nil, fmt.Errorf("failed to get cluster master maintenance policy")
	}

	if err := h.flattenMasterMaintenancePolicy(p); err != nil {
		return nil, err
	}

	switch m := clusterMaster.GetMasterType().(type) {
	case *k8s.Master_ZonalMaster:
		h.flattenClusterZonalMaster(m)
	case *k8s.Master_RegionalMaster:
		h.flattenClusterRegionalMaster(m)
	default:
		return nil, fmt.Errorf("unsupported Kubernetes master type (currently only zonal and regional master are supported)")
	}

	versionInfo := clusterMaster.GetVersionInfo()
	if versionInfo == nil {
		return nil, fmt.Errorf("failed to get Kubernetes master version info")
	}

	h.versionInfo["current_version"] = versionInfo.GetCurrentVersion()
	h.versionInfo["new_revision_available"] = versionInfo.GetNewRevisionAvailable()
	h.versionInfo["new_revision_summary"] = versionInfo.GetNewRevisionSummary()
	h.versionInfo["version_deprecated"] = versionInfo.GetVersionDeprecated()
	return h, nil
}

func dayOfWeekHash(v interface{}) int {
	window, err := expandDayMaintenanceWindow(v.(map[string]interface{}))
	if err != nil {
		return 0
	}

	hashString := fmt.Sprintf("%s-%s-%s",
		strings.ToLower(window.day.String()),
		formatTimeOfDay(window.startTime),
		formatDuration(window.duration),
	)

	return hashcode.String(hashString)
}

func parseDayOfWeek(v string) (dayofweek.DayOfWeek, error) {
	upper := strings.ToUpper(v)
	val, ok := dayofweek.DayOfWeek_value[upper]

	// do not allow DAY_OF_WEEK_UNSPECIFIED here
	if !ok || val == 0 {
		return dayofweek.DayOfWeek(0), fmt.Errorf("value for 'day' should be one of %s (any case), not `%s`",
			getJoinedKeys(stringSliceToLower(getEnumValueMapKeysExt(dayofweek.DayOfWeek_value, true))), v)
	}
	return dayofweek.DayOfWeek(val), nil
}

func shouldSuppressDiffForDayOfWeek(k, old, new string, d *schema.ResourceData) bool {
	return strings.EqualFold(old, new)
}

func shouldSuppressDiffForTimeOfDay(k, old, new string, d *schema.ResourceData) bool {
	t1, err := parseDayTime(old)
	if err != nil {
		return false
	}

	t2, err := parseDayTime(new)
	if err != nil {
		return false
	}

	return formatTimeOfDay(t1) == formatTimeOfDay(t2)
}

func formatTimeOfDay(ts *timeofday.TimeOfDay) string {
	tt := time.Date(0, 0, 0, int(ts.GetHours()), int(ts.GetMinutes()), int(ts.GetSeconds()), int(ts.GetNanos()), time.UTC)
	return tt.Format("15:04:05.000000000")
}

func parseDuration(s string) (*duration.Duration, error) {
	if s == "" {
		return nil, nil
	}

	v, err := time.ParseDuration(s)
	if err != nil {
		return nil, fmt.Errorf("failed to parse duration: %v", err)
	}

	if v < 0 {
		return nil, fmt.Errorf("can not use negative duration")
	}

	return ptypes.DurationProto(v), nil
}

func formatDuration(d *duration.Duration) string {
	if d == nil {
		return ""
	}

	td := time.Nanosecond*time.Duration(d.GetNanos()) + time.Second*time.Duration(d.GetSeconds())
	return td.String()
}

func shouldSuppressDiffForTimeDuration(k, old, new string, d *schema.ResourceData) bool {
	d1, err := parseDuration(old)
	if err != nil {
		return false
	}

	d2, err := parseDuration(new)
	if err != nil {
		return false
	}

	if d1 == nil && d2 == nil {
		return true
	}

	if d1 != nil && d2 != nil {
		return d1.Seconds == d2.Seconds && d1.Nanos == d2.Nanos
	}

	return false
}

func parseDayTime(s string) (*timeofday.TimeOfDay, error) {
	formats := []string{"15:04:05.000000000", "15:04:05", "15:04"}

	var ts time.Time
	var err error
	for _, f := range formats {
		if ts, err = time.ParseInLocation(f, s, time.UTC); err == nil {
			break
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse time of day. Expected HH:MM:SS or HH:MM, got: %s", s)
	}

	return &timeofday.TimeOfDay{
		Hours:   int32(ts.Hour()),
		Minutes: int32(ts.Minute()),
		Seconds: int32(ts.Second()),
		Nanos:   int32(ts.Nanosecond()),
	}, nil
}
