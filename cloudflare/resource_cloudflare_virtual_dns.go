package cloudflare

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/cloudflare/cloudflare-go"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/pkg/errors"
)

func resourceCloudflareVirtualDNS() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudflareVirtualDNSCreate,
		Read:   resourceCloudflareVirtualDNSRead,
		Update: resourceCloudflareVirtualDNSUpdate,
		Delete: resourceCloudflareVirtualDNSDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringLenBetween(1, 160),
			},
			"origin_ips": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.SingleIP(),
				},
			},
			"virtual_dns_ips": {
				Type:     schema.TypeSet,
				Computed: true,
				Optional: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.SingleIP(),
				},
			},
			"minimum_cache_ttl": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      60,
				ValidateFunc: validation.IntBetween(30, 36000),
			},
			"maximum_cache_ttl": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      900,
				ValidateFunc: validation.IntBetween(30, 36000),
			},
			"deprecate_any_requests": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"ecs_fallback": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"ratelimit": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      5000,
				ValidateFunc: validation.IntBetween(0, 100000000),
			},
		},
	}
}

func resourceCloudflareVirtualDNSCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.API)

	virtualDNS := &cloudflare.VirtualDNS{
		Name:            d.Get("name").(string),
		OriginIPs:       expandInterfaceToStringList(d.Get("origin_ips").(*schema.Set).List()),
		MinimumCacheTTL: d.Get("minimum_cache_ttl").(uint),
		MaximumCacheTTL: d.Get("maximum_cache_ttl").(uint),
	}
	if val, ok := d.GetOk("deprecate_any_requests"); ok {
		virtualDNS.DeprecateAnyRequests = val.(bool)
	}
	if val, ok := d.GetOk("ecs_fallback"); ok {
		virtualDNS.EcsFallback = val.(bool)
	}
	if val, ok := d.GetOk("ratelimit"); ok {
		virtualDNS.RateLimit = val.(uint)
	}

	log.Printf("[DEBUG] Creating Cloudflare VirtualDNS from struct: %+v", virtualDNS)

	res, err := client.CreateAccountVirtualDNS(context.Background(), client.AccountID, virtualDNS)
	if err != nil {
		return errors.Wrap(err, "error creating virtual dns")
	}

	if res.ID == "" {
		return fmt.Errorf("cailed to find id in create response; resource was empty")
	}

	d.SetId(res.ID)

	log.Printf("[INFO] New Cloudflare VirtualDNS created with  ID: %s", d.Id())

	return resourceCloudflareVirtualDNSRead(d, meta)
}

func resourceCloudflareVirtualDNSUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.API)

	virtualDNS := &cloudflare.VirtualDNS{
		ID:              d.Id(),
		Name:            d.Get("name").(string),
		OriginIPs:       expandInterfaceToStringList(d.Get("origin_ips").(*schema.Set).List()),
		MinimumCacheTTL: d.Get("minimum_cache_ttl").(uint),
		MaximumCacheTTL: d.Get("maximum_cache_ttl").(uint),
	}
	if val, ok := d.GetOk("virtual_dns_ips"); ok {
		virtualDNS.VirtualDNSIPs = expandInterfaceToStringList(val.(*schema.Set).List())
	}
	if val, ok := d.GetOk("deprecate_any_requests"); ok {
		virtualDNS.DeprecateAnyRequests = val.(bool)
	}
	if val, ok := d.GetOk("ecs_fallback"); ok {
		virtualDNS.EcsFallback = val.(bool)
	}
	if val, ok := d.GetOk("ratelimit"); ok {
		virtualDNS.RateLimit = val.(uint)
	}

	log.Printf("[DEBUG] Updating Cloudflare VirtualDNS from struct: %+v", virtualDNS)

	err := client.UpdateAccountVirtualDNS(context.Background(), client.AccountID, d.Id(), virtualDNS)
	if err != nil {
		return errors.Wrap(err, "error updating VirtualDNS")
	}

	return resourceCloudflareVirtualDNSRead(d, meta)
}

func resourceCloudflareVirtualDNSRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.API)

	virtualDNS, err := client.AccountVirtualDNS(context.Background(), client.AccountID, d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "HTTP status 404") {
			log.Printf("[INFO] VirtualDNS %s no longer exists", d.Id())
			d.SetId("")
			return nil
		} else {
			msg := fmt.Sprintf("Error reading VirtualDNS from API for resource %s", d.Id())
			return errors.Wrap(err, msg)
		}
	}

	log.Printf("[DEBUG] Read VirtualDNSfrom API as struct: %+v", virtualDNS)

	d.Set("name", virtualDNS.Name)
	d.Set("origin_ips", schema.NewSet(schema.HashString, flattenStringList(virtualDNS.OriginIPs)))
	d.Set("virtual_dns_ips", schema.NewSet(schema.HashString, flattenStringList(virtualDNS.VirtualDNSIPs)))
	d.Set("minimum_cache_ttl", virtualDNS.MinimumCacheTTL)
	d.Set("maximum_cache_ttl", virtualDNS.MaximumCacheTTL)
	d.Set("deprecate_any_requests", virtualDNS.DeprecateAnyRequests)
	d.Set("ecs_fallback", virtualDNS.EcsFallback)
	d.Set("ratelimit", virtualDNS.RateLimit)

	return nil
}

func resourceCloudflareVirtualDNSDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.API)

	log.Printf("[INFO] Deleting Cloudflare VirtualDNS: %s", d.Id())

	err := client.DeleteAccountVirtualDNS(context.Background(), client.AccountID, d.Id())
	if err != nil {
		return errors.Wrap(err, "error deleting Cloudflare VirtualDNS")
	}

	return nil
}
