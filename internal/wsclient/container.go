package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for LXC containers: container.{...}.
//
// The whole namespace is new in TrueNAS 26.0. On 25.10 and older it does
// not exist, so every call here answers method-not-found. That is
// translated into a message naming the required version rather than left
// as a bare -32601, which names only the method and points nowhere near
// the cause.
//
// container.create, container.delete and container.stop are @job upstream:
// they return a job ID and the work happens asynchronously, so they go
// through CallJob. container.start and the reads do not.

// errContainerUnsupported turns a method-not-found on this namespace into
// an actionable diagnostic.
//
// Only method-not-found is translated. Any other failure keeps its own
// error, or a genuine permission or validation problem would be reported
// as a version problem.
func errContainerUnsupported(err error, what string) error {
	if !isMethodUnknown(err) {
		return err
	}
	return fmt.Errorf("%s: containers require TrueNAS 26.0 or newer. "+
		"This server has no container namespace, so the truenas_container "+
		"resource cannot be used against it (%w)", what, err)
}

// GetContainer retrieves a container by ID.
func (c *Client) GetContainer(ctx context.Context, id int) (*types.Container, error) {
	tflog.Trace(ctx, "GetContainer (ws) start")

	result, err := c.Call(ctx, "container.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, errContainerUnsupported(err, fmt.Sprintf("getting container %d", id))
	}

	var container types.Container
	if err := json.Unmarshal(result, &container); err != nil {
		return nil, fmt.Errorf("parsing container response: %w", err)
	}

	tflog.Trace(ctx, "GetContainer (ws) success")
	return &container, nil
}

// ListContainers retrieves all containers.
func (c *Client) ListContainers(ctx context.Context) ([]types.Container, error) {
	tflog.Trace(ctx, "ListContainers (ws) start")

	result, err := c.Call(ctx, "container.query", nil,
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, errContainerUnsupported(err, "listing containers")
	}

	var containers []types.Container
	if err := json.Unmarshal(result, &containers); err != nil {
		return nil, fmt.Errorf("parsing container list: %w", err)
	}

	tflog.Trace(ctx, "ListContainers (ws) success")
	return containers, nil
}

// CreateContainer creates a container.
//
// container.create is a job: it pulls the image and builds the root
// filesystem, which takes far longer than a plain call. CallJob waits for
// the terminal state, and the job result is the new container entry.
func (c *Client) CreateContainer(ctx context.Context, req *types.ContainerCreateRequest) (*types.Container, error) {
	tflog.Trace(ctx, "CreateContainer (ws) start")

	result, err := c.CallJob(ctx, "container.create",
		[]interface{}{req}, CallOptions{Idempotent: false}, 0)
	if err != nil {
		return nil, errContainerUnsupported(err, fmt.Sprintf("creating container %q", req.Name))
	}

	var container types.Container
	if err := json.Unmarshal(result, &container); err != nil {
		return nil, fmt.Errorf("parsing container create response: %w", err)
	}

	tflog.Trace(ctx, "CreateContainer (ws) success")
	return &container, nil
}

// UpdateContainer updates a container.
func (c *Client) UpdateContainer(ctx context.Context, id int, req *types.ContainerUpdateRequest) (*types.Container, error) {
	tflog.Trace(ctx, "UpdateContainer (ws) start")

	result, err := c.Call(ctx, "container.update",
		[]interface{}{id, req}, CallOptions{Idempotent: false})
	if err != nil {
		return nil, errContainerUnsupported(err, fmt.Sprintf("updating container %d", id))
	}

	var container types.Container
	if err := json.Unmarshal(result, &container); err != nil {
		return nil, fmt.Errorf("parsing container update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateContainer (ws) success")
	return &container, nil
}

// DeleteContainer removes a container.
//
// opts.Recursive is destructive well beyond the container itself (child
// datasets, snapshots, clones elsewhere in the pool, holds), so callers
// pass it explicitly rather than getting it by default.
//
// container.delete changed shape during the 26.0 cycle. 26.0-BETA.1 takes
// the id alone and returns directly; later builds take a second options
// argument and run as a job. The newer form is tried first, and a server
// that rejects the extra argument gets the older one. Argument arity is
// checked before the method body runs, so the first attempt cannot have
// deleted anything.
func (c *Client) DeleteContainer(ctx context.Context, id int, opts *types.ContainerDeleteOptions) error {
	tflog.Trace(ctx, "DeleteContainer (ws) start")

	if opts == nil {
		opts = &types.ContainerDeleteOptions{}
	}
	_, err := c.CallJob(ctx, "container.delete",
		[]interface{}{id, opts}, CallOptions{Idempotent: false}, 0)
	if err != nil && isTooManyArguments(err) {
		tflog.Debug(ctx, "container.delete takes no options on this server, falling back to the single-argument form",
			map[string]interface{}{"id": id})
		// The single-argument form is not a job, so this must not poll.
		// It also has no force flag: a running container is stopped by
		// TrueNAS itself, or the delete fails and the error is reported.
		_, err = c.Call(ctx, "container.delete", []interface{}{id}, CallOptions{Idempotent: false})
	}
	if err != nil {
		return errContainerUnsupported(err, fmt.Sprintf("deleting container %d", id))
	}

	tflog.Trace(ctx, "DeleteContainer (ws) success")
	return nil
}

// StartContainer starts a container. Unlike create/delete/stop this is not
// a job upstream.
func (c *Client) StartContainer(ctx context.Context, id int) error {
	tflog.Trace(ctx, "StartContainer (ws) start")

	if _, err := c.Call(ctx, "container.start",
		[]interface{}{id}, CallOptions{Idempotent: false}); err != nil {
		return errContainerUnsupported(err, fmt.Sprintf("starting container %d", id))
	}

	tflog.Trace(ctx, "StartContainer (ws) success")
	return nil
}

// StopContainer stops a container.
func (c *Client) StopContainer(ctx context.Context, id int, opts *types.ContainerStopOptions) error {
	tflog.Trace(ctx, "StopContainer (ws) start")

	if opts == nil {
		opts = &types.ContainerStopOptions{}
	}
	if _, err := c.CallJob(ctx, "container.stop",
		[]interface{}{id, opts}, CallOptions{Idempotent: false}, 0); err != nil {
		return errContainerUnsupported(err, fmt.Sprintf("stopping container %d", id))
	}

	tflog.Trace(ctx, "StopContainer (ws) success")
	return nil
}

// GetContainerPoolChoices lists the pools a container root filesystem can
// live on, keyed by pool name.
func (c *Client) GetContainerPoolChoices(ctx context.Context) (map[string]string, error) {
	tflog.Trace(ctx, "GetContainerPoolChoices (ws) start")

	result, err := c.Call(ctx, "container.pool_choices", nil,
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, errContainerUnsupported(err, "listing container pool choices")
	}

	var choices map[string]string
	if err := json.Unmarshal(result, &choices); err != nil {
		return nil, fmt.Errorf("parsing container pool choices: %w", err)
	}

	tflog.Trace(ctx, "GetContainerPoolChoices (ws) success")
	return choices, nil
}
