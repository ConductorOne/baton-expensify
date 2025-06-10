package expensify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
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
	NextPage     string   `json:"nextPage"`
}

type PolicyResponse struct {
	PolicyInfo   map[string]Employees `json:"policyInfo"`
	ResponseCode int64                `json:"responseCode"`
	NextPage     string               `json:"nextPage"`
}

type Error struct {
	Message    string `json:"responseMessage"`
	StatusCode int    `json:"responseCode"`
}

// GetPolicies returns policies that user is an admin of.
func (c *Client) GetPolicies(ctx context.Context) ([]Policy, error) {
	var allPolicies []Policy
	nextPage := "initial"

	for nextPage != "" {
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
		rl := &v2.RateLimitDescription{}
		err := c.doRequest(ctx, body, &res, rl)
		if err != nil {
			return nil, err
		}

		allPolicies = append(allPolicies, res.PolicyList...)

		// For testing: always return a next page to force continuous API calls. TODO [mb] remove this.
		nextPage = "next-page-token"
	}

	return allPolicies, nil
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
	rl := &v2.RateLimitDescription{}
	err := c.doRequest(ctx, body, &res, rl)
	if err != nil {
		return nil, err
	}

	return res.PolicyInfo[policyId].Employees, nil
}

func (c *Client) doRequest(ctx context.Context, body interface{}, resType interface{}, rl *v2.RateLimitDescription) error {
	strBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	data := url.Values{}
	data.Set("requestJobDescription", string(strBody))

	apiURL, err := url.Parse(BaseUrl)
	if err != nil {
		return fmt.Errorf("failed to parse base URL: %w", err)
	}

	req, err := c.httpClient.NewRequest(
		ctx,
		http.MethodPost,
		apiURL,
		uhttp.WithContentTypeFormHeader(),
		uhttp.WithFormBody(data.Encode()),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req, uhttp.WithAlwaysJSONResponse(&resType), uhttp.WithRatelimitData(rl))
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	return nil
}
