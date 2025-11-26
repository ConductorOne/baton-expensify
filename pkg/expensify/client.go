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

	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"google.golang.org/grpc/codes"
)

const BaseUrl = "https://integrations.expensify.com/Integration-Server/ExpensifyIntegrations"

type Client struct {
	httpClient        *uhttp.BaseHttpClient
	partnerUserID     string
	partnerUserSecret string
}

func NewClient(ctx context.Context, partnerUserID string, partnerUserSecret string) (*Client, error) {
	httpClient, err := uhttp.NewClient(ctx, uhttp.WithLogger(true, ctxzap.Extract(ctx)))
	if err != nil {
		return nil, fmt.Errorf("failed to create http client: %w", err)
	}

	return &Client{
		partnerUserID:     partnerUserID,
		partnerUserSecret: partnerUserSecret,
		httpClient:        uhttp.NewBaseHttpClient(httpClient),
	}, nil
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

	var res PolicyListResponse
	err := c.doRequest(ctx, body, &res)
	if err != nil {
		return nil, err
	}

	return res.PolicyList, nil
}

// GetPolicyEmployees returns employees for a single policy.
func (c *Client) GetPolicyEmployees(ctx context.Context, policyId string) ([]User, error) {
	var fields, policyIDs []string
	fields = append(fields, "employees")
	policyIDs = append(policyIDs, policyId)
	body := PolicyRequestBody{
		Type: "get",
		Credentials: Credentials{
			PartnerUserID:     c.partnerUserID,
			PartnerUserSecret: c.partnerUserSecret,
		},
		InputSettings: PolicyInputSettings{
			Type:         "policy",
			Fields:       fields,
			PolicyIDList: policyIDs,
		},
	}

	var res PolicyResponse
	err := c.doRequest(ctx, body, &res)
	if err != nil {
		return nil, err
	}

	return res.PolicyInfo[policyId].Employees, nil
}

func (c *Client) doRequest(ctx context.Context, body interface{}, resType interface{}) error {
	strBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	data := url.Values{}
	data.Set("requestJobDescription", string(strBody))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, BaseUrl, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var (
		buf bytes.Buffer
		r   = io.TeeReader(resp.Body, &buf)
	)

	var errResp Error
	if err = json.NewDecoder(r).Decode(&errResp); err != nil {
		return err
	}

	// Expensify always returns HTTP 200, but uses responseCode in the JSON body for errors.
	// See: https://integrations.expensify.com/Integration-Server/doc/#basic-format
	if code := errResp.StatusCode; code != 0 && code != http.StatusOK {
		apiErr := fmt.Errorf("%s", errResp.Message)
		msgLower := strings.ToLower(errResp.Message)

		// Priority 1: Check for authentication errors by message content.
		// Expensify returns "Authentication error" (responseCode 404) for invalid credentials.
		if strings.Contains(msgLower, "authentication error") {
			return uhttp.WrapErrors(codes.Unauthenticated, fmt.Sprintf("expensify-connector: %s", errResp.Message), apiErr)
		}

		// Priority 2: Classify by Expensify's responseCode.
		switch code {
		case 401:
			// Unauthorized - authentication required
			return uhttp.WrapErrors(codes.Unauthenticated, fmt.Sprintf("expensify-connector: %s", errResp.Message), apiErr)
		case 403:
			// Forbidden - user is not an admin of the policy
			return uhttp.WrapErrors(codes.PermissionDenied, fmt.Sprintf("expensify-connector: %s", errResp.Message), apiErr)
		case 404:
			// Expensify uses 404 for authentication errors (already handled above) and not found
			return uhttp.WrapErrors(codes.NotFound, fmt.Sprintf("expensify-connector: %s", errResp.Message), apiErr)
		case 410:
			// Validation error - invalid policyID, missing required parameters, etc.
			// Common messages: "'xxx' is not a valid policy ID", "Required parameter 'policyIDList' is missing"
			return uhttp.WrapErrors(codes.InvalidArgument, fmt.Sprintf("expensify-connector: %s", errResp.Message), apiErr)
		case 429:
			// Rate limit exceeded (5 requests/10s or 20 requests/60s per documentation)
			return uhttp.WrapErrors(codes.ResourceExhausted, fmt.Sprintf("expensify-connector: %s", errResp.Message), apiErr)
		case 500:
			// Internal server error from Expensify - treat as service unavailable
			return uhttp.WrapErrors(codes.Unavailable, fmt.Sprintf("expensify-connector: %s", errResp.Message), apiErr)
		case 503:
			// Service unavailable
			return uhttp.WrapErrors(codes.Unavailable, fmt.Sprintf("expensify-connector: %s", errResp.Message), apiErr)
		default:
			// Unknown error code
			return uhttp.WrapErrors(codes.Unknown, fmt.Sprintf("expensify-connector: %s", errResp.Message), apiErr)
		}
	}

	if err := json.NewDecoder(&buf).Decode(resType); err != nil {
		return err
	}

	return nil
}
