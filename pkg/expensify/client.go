package expensify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const BaseUrl = "https://integrations.expensify.com/Integration-Server/ExpensifyIntegrations"

type Client struct {
	httpClient        *http.Client
	partnerUserID     string
	partnerUserSecret string
}

func NewClient(partnerUserID string, partnerUserSecret string, httpClient *http.Client) *Client {
	return &Client{
		partnerUserID:     partnerUserID,
		partnerUserSecret: partnerUserSecret,
		httpClient:        httpClient,
	}
}

type Credentials struct {
	PartnerUserID     string `json:"partnerUserID"`
	PartnerUserSecret string `json:"partnerUserSecret"`
}

type PolicyInputSettings struct {
	Type         string   `json:"type"`
	Fields       []string `json:"fields,omitempty"`
	PolicyIDList []string `json:"policyIDList"`
	UserEmail    string   `json:"userEmail,omitempty"`
}

type PoliciesInputSettings struct {
	Type      string `json:"type"`
	AdminOnly bool   `json:"adminOnly,omitempty"`
	UserEmail string `json:"userEmail,omitempty"`
}

type PolicyRequestBody struct {
	Type          string              `json:"type"`
	Credentials   Credentials         `json:"credentials"`
	InputSettings PolicyInputSettings `json:"inputSettings"`
}

type PoliciesRequestBody struct {
	Type          string                `json:"type"`
	Credentials   Credentials           `json:"credentials"`
	InputSettings PoliciesInputSettings `json:"inputSettings"`
}

type PolicyListResponse struct {
	PolicyList   []Policy `json:"policyList"`
	ResponseCode int64    `json:"responseCode"`
}

type Employees struct {
	Employees []User `json:"employees"`
}

type PolicyResponse struct {
	PolicyInfo   map[string]Employees `json:"policyInfo"`
	ResponseCode int64                `json:"responseCode"`
}

type Error struct {
	Message    string `json:"responseMessage"`
	StatusCode int    `json:"responseCode"`
}

// GetPolicies returns policies that user is an admin of.
func (c *Client) GetPolicies(ctx context.Context) ([]Policy, error) {
	body := PoliciesRequestBody{
		Type: "get",
		Credentials: Credentials{
			PartnerUserID:     c.partnerUserID,
			PartnerUserSecret: c.partnerUserSecret,
		},
		InputSettings: PoliciesInputSettings{
			Type:      "policyList",
			AdminOnly: true,
		},
	}

	strBody, _ := json.Marshal(body)
	data := url.Values{}
	data.Set("requestJobDescription", string(strBody))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, BaseUrl, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var (
		buf bytes.Buffer
		r   = io.TeeReader(resp.Body, &buf)
	)

	var errResp Error
	var res PolicyListResponse
	if err = json.NewDecoder(r).Decode(&errResp); err != nil {
		return nil, err
	} else if code := errResp.StatusCode; code != 0 && code != http.StatusOK {
		return nil, fmt.Errorf("error: %s", errResp.Message)
	}

	if err := json.NewDecoder(&buf).Decode(&res); err != nil {
		return nil, err
	}
	return res.PolicyList, nil
}

// GetPolicyEmployees returns employees for a signle policy.
func (c *Client) GetPolicyEmployees(ctx context.Context, policyId string) ([]User, error) {
	var fields, policyIds []string
	fields = append(fields, "employees")
	policyIds = append(policyIds, policyId)
	body := PolicyRequestBody{
		Type: "get",
		Credentials: Credentials{
			PartnerUserID:     c.partnerUserID,
			PartnerUserSecret: c.partnerUserSecret,
		},
		InputSettings: PolicyInputSettings{
			Type:         "policy",
			Fields:       fields,
			PolicyIDList: policyIds,
		},
	}

	strBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	data := url.Values{}
	data.Set("requestJobDescription", string(strBody))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, BaseUrl, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var (
		buf bytes.Buffer
		r   = io.TeeReader(resp.Body, &buf)
	)

	var errResp Error
	var res PolicyResponse
	if err = json.NewDecoder(r).Decode(&errResp); err != nil {
		return nil, err
	} else if code := errResp.StatusCode; code != 0 && code != http.StatusOK {
		return nil, fmt.Errorf("error: %s", errResp.Message)
	}

	if err := json.NewDecoder(&buf).Decode(&res); err != nil {
		return nil, err
	}

	return res.PolicyInfo[policyId].Employees, nil
}
