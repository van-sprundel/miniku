package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"miniku/pkg/types"
	"net/http"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New(apiServerURL string) *Client {
	return &Client{
		baseURL:    apiServerURL,
		httpClient: &http.Client{},
	}
}

// Pods

func (c *Client) ListPods() ([]types.Pod, error) {
	var pods []types.Pod
	if err := c.list("/pods", &pods); err != nil {
		return nil, err
	}
	return pods, nil
}

func (c *Client) GetPod(name string) (types.Pod, bool, error) {
	var pod types.Pod
	found, err := c.get("/pods/"+name, &pod)
	return pod, found, err
}

func (c *Client) CreatePod(pod types.Pod) error {
	return c.create("/pods", pod)
}

func (c *Client) UpdatePod(name string, pod types.Pod) error {
	return c.update("/pods/"+name, pod)
}

func (c *Client) DeletePod(name string) error {
	return c.delete("/pods/" + name)
}

func (c *Client) ListReplicaSets() ([]types.ReplicaSet, error) {
	var rsList []types.ReplicaSet
	if err := c.list("/replicasets", &rsList); err != nil {
		return nil, err
	}
	return rsList, nil
}

func (c *Client) GetReplicaSet(name string) (types.ReplicaSet, bool, error) {
	var rs types.ReplicaSet
	found, err := c.get("/replicasets/"+name, &rs)
	return rs, found, err
}

func (c *Client) CreateReplicaSet(rs types.ReplicaSet) error {
	return c.create("/replicasets", rs)
}

func (c *Client) UpdateReplicaSet(name string, rs types.ReplicaSet) error {
	return c.update("/replicasets/"+name, rs)
}

func (c *Client) DeleteReplicaSet(name string) error {
	return c.delete("/replicasets/" + name)
}

func (c *Client) ListNodes() ([]types.Node, error) {
	var nodes []types.Node
	if err := c.list("/nodes", &nodes); err != nil {
		return nil, err
	}
	return nodes, nil
}

func (c *Client) GetNode(name string) (types.Node, bool, error) {
	var node types.Node
	found, err := c.get("/nodes/"+name, &node)
	return node, found, err
}

func (c *Client) CreateNode(node types.Node) error {
	return c.create("/nodes", node)
}

func (c *Client) UpdateNode(name string, node types.Node) error {
	return c.update("/nodes/"+name, node)
}

func (c *Client) DeleteNode(name string) error {
	return c.delete("/nodes/" + name)
}

func (c *Client) list(path string, out any) error {
	resp, err := c.httpClient.Get(c.baseURL + path)
	if err != nil {
		return fmt.Errorf("GET %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: status %d", path, resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) get(path string, out any) (bool, error) {
	resp, err := c.httpClient.Get(c.baseURL + path)
	if err != nil {
		return false, fmt.Errorf("GET %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("GET %s: status %d", path, resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return false, err
	}
	return true, nil
}

func (c *Client) create(path string, body any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Post(c.baseURL+path, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("POST %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("POST %s: status %d", path, resp.StatusCode)
	}
	return nil
}

func (c *Client) update(path string, body any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPut, c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("PUT %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("PUT %s: status %d", path, resp.StatusCode)
	}
	return nil
}

func (c *Client) delete(path string) error {
	req, err := http.NewRequest(http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("DELETE %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("DELETE %s: status %d", path, resp.StatusCode)
	}
	return nil
}
