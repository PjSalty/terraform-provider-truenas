package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for container devices:
// container.device.{...}. New in TrueNAS 26.0 along with the container
// namespace, and absent on 25.10 and older, where every call here answers
// method-not-found. That is translated into a message naming the required
// version rather than left as a bare -32601.
//
// None of these are jobs upstream, unlike container.create.

// errContainerDeviceUnsupported turns a method-not-found on this namespace
// into an actionable diagnostic. Only method-not-found is translated, so a
// permission or validation failure keeps its own error.
func errContainerDeviceUnsupported(err error, what string) error {
	if !isMethodUnknown(err) {
		return err
	}
	return fmt.Errorf("%s: container devices require TrueNAS 26.0 or newer. "+
		"This server has no container.device namespace, so the "+
		"truenas_container_device resource cannot be used against it (%w)", what, err)
}

// GetContainerDevice retrieves a container device by ID.
func (c *Client) GetContainerDevice(ctx context.Context, id int) (*types.ContainerDevice, error) {
	tflog.Trace(ctx, "GetContainerDevice (ws) start")

	result, err := c.Call(ctx, "container.device.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, errContainerDeviceUnsupported(err, fmt.Sprintf("getting container device %d", id))
	}

	var device types.ContainerDevice
	if err := json.Unmarshal(result, &device); err != nil {
		return nil, fmt.Errorf("parsing container device response: %w", err)
	}

	tflog.Trace(ctx, "GetContainerDevice (ws) success")
	return &device, nil
}

// ListContainerDevices retrieves all container devices.
func (c *Client) ListContainerDevices(ctx context.Context) ([]types.ContainerDevice, error) {
	tflog.Trace(ctx, "ListContainerDevices (ws) start")

	result, err := c.Call(ctx, "container.device.query", nil,
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, errContainerDeviceUnsupported(err, "listing container devices")
	}

	var devices []types.ContainerDevice
	if err := json.Unmarshal(result, &devices); err != nil {
		return nil, fmt.Errorf("parsing container device list: %w", err)
	}

	tflog.Trace(ctx, "ListContainerDevices (ws) success")
	return devices, nil
}

// CreateContainerDevice attaches a device to a container.
func (c *Client) CreateContainerDevice(ctx context.Context, req *types.ContainerDeviceCreateRequest) (*types.ContainerDevice, error) {
	tflog.Trace(ctx, "CreateContainerDevice (ws) start")

	result, err := c.Call(ctx, "container.device.create",
		[]interface{}{req}, CallOptions{Idempotent: false})
	if err != nil {
		return nil, errContainerDeviceUnsupported(err,
			fmt.Sprintf("creating device on container %d", req.Container))
	}

	var device types.ContainerDevice
	if err := json.Unmarshal(result, &device); err != nil {
		return nil, fmt.Errorf("parsing container device create response: %w", err)
	}

	tflog.Trace(ctx, "CreateContainerDevice (ws) success")
	return &device, nil
}

// UpdateContainerDevice updates a container device.
func (c *Client) UpdateContainerDevice(ctx context.Context, id int, req *types.ContainerDeviceUpdateRequest) (*types.ContainerDevice, error) {
	tflog.Trace(ctx, "UpdateContainerDevice (ws) start")

	result, err := c.Call(ctx, "container.device.update",
		[]interface{}{id, req}, CallOptions{Idempotent: false})
	if err != nil {
		return nil, errContainerDeviceUnsupported(err, fmt.Sprintf("updating container device %d", id))
	}

	var device types.ContainerDevice
	if err := json.Unmarshal(result, &device); err != nil {
		return nil, fmt.Errorf("parsing container device update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateContainerDevice (ws) success")
	return &device, nil
}

// DeleteContainerDevice detaches a device from its container.
//
// opts.RawFile and opts.Zvol destroy the storage behind the device, so
// callers pass them explicitly rather than getting them by default.
func (c *Client) DeleteContainerDevice(ctx context.Context, id int, opts *types.ContainerDeviceDeleteOptions) error {
	tflog.Trace(ctx, "DeleteContainerDevice (ws) start")

	if opts == nil {
		opts = &types.ContainerDeviceDeleteOptions{}
	}
	if _, err := c.Call(ctx, "container.device.delete",
		[]interface{}{id, opts}, CallOptions{Idempotent: false}); err != nil {
		return errContainerDeviceUnsupported(err, fmt.Sprintf("deleting container device %d", id))
	}

	tflog.Trace(ctx, "DeleteContainerDevice (ws) success")
	return nil
}

// GetContainerNICAttachChoices lists the interfaces a NIC device can
// attach to, split into bridges and MACVLAN parents.
func (c *Client) GetContainerNICAttachChoices(ctx context.Context) (*types.ContainerNICAttachChoices, error) {
	tflog.Trace(ctx, "GetContainerNICAttachChoices (ws) start")

	result, err := c.Call(ctx, "container.device.nic_attach_choices", nil,
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, errContainerDeviceUnsupported(err, "listing NIC attach choices")
	}

	var choices types.ContainerNICAttachChoices
	if err := json.Unmarshal(result, &choices); err != nil {
		return nil, fmt.Errorf("parsing NIC attach choices: %w", err)
	}

	tflog.Trace(ctx, "GetContainerNICAttachChoices (ws) success")
	return &choices, nil
}

// ListContainerImages returns every image the LXC registry currently
// publishes, with its available versions.
//
// This reaches out to the upstream image registry, so it is slower than a
// local read and fails when the server has no route to it.
func (c *Client) ListContainerImages(ctx context.Context) ([]types.ContainerImage, error) {
	tflog.Trace(ctx, "ListContainerImages (ws) start")

	result, err := c.Call(ctx, "container.image.query_registry", nil,
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, errContainerDeviceUnsupported(err, "listing container images")
	}

	var images []types.ContainerImage
	if err := json.Unmarshal(result, &images); err != nil {
		return nil, fmt.Errorf("parsing container image list: %w", err)
	}

	tflog.Trace(ctx, "ListContainerImages (ws) success")
	return images, nil
}
