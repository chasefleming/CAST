package shared

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog/log"
)

type SnapshotClient struct {
	BaseURL    string
	HTTPClient *http.Client
	Env        string
	Fa         FlowAdapter
}

type Snapshot struct {
	ID           string    `json:"id"`
	Block_height uint64    `json:"blockHeight"`
	Started      time.Time `json:"started"`
	Finished     time.Time `json:"finished"`
}

type LatestBlockHeight struct {
	ID              string    `json:"id"`
	FungibleTokenID string    `json:"fungibleTokenId"`
	BlockHeight     uint64    `json:"blockHeight"`
	Started         time.Time `json:"started"`
	Finished        time.Time `json:"finished"`
	Status          string    `json:"status"`
	Attempts        int       `json:"attempts"`
}

type balanceAtBlockheight struct {
	ID              string    `json:"id"`
	FungibleTokenID string    `json:"fungibleTokenId"`
	Addr            string    `json:"address"`
	Balance         uint64    `json:"balance"`
	BlockHeight     uint64    `json:"blockHeight"`
	CreatedAt       time.Time `json:"createdAt"`
}

type SnapshotResponse struct {
	Data SnapshotData `json:"data"`
}

type SnapshotData struct {
	Message     string `json:"message"`
	Status      string `json:"status"`
	BlockHeight uint64 `json:"blockHeight"`
}

type FungibleTokenContract struct {
	ContractAddress      string `json:"contractAddress"`
	ContractName         string `json:"contractName"`
	PublicCapabilityPath string `json:"publicCapabilityPath"`
}

var (
	DummySnapshot = Snapshot{
		ID:           "1",
		Block_height: 0,
		Started:      time.Now(),
		Finished:     time.Now(),
	}

	DummyBalance = FTBalanceResponse{
		PrimaryAccountBalance:   100,
		SecondaryAccountBalance: 100,
		StakingBalance:          100,
		BlockHeight:             0,
	}
)

func NewSnapshotClient(baseUrl string, fa FlowAdapter) *SnapshotClient {
	return &SnapshotClient{
		BaseURL: baseUrl,
		HTTPClient: &http.Client{
			Timeout: time.Second * 10,
		},
		Env: os.Getenv("APP_ENV"),
		Fa:  fa,
	}
}

func (c *SnapshotClient) TakeSnapshot(contract Contract) (*SnapshotResponse, error) {
	if c.bypass() {
		return &SnapshotResponse{
			Data: SnapshotData{
				Message:     "",
				Status:      "success",
				BlockHeight: 1000000,
			},
		}, nil
	}

	var r *SnapshotResponse = &SnapshotResponse{}

	url := c.setSnapshotUrl(contract, "take-snapshot")
	log.Info().Msgf("Taking token snapshot. Url: %s", url)
	req, err := c.setRequestMethod("POST", url, nil)
	if err != nil {
		log.Debug().Err(err).Msg("SnapshotClient TakeSnapshot request error")
		return r, err
	}

	if _, err := c.sendRequest(req, r); err != nil {
		log.Debug().Err(err).Msg("SnapshotClient takeSnapshot request error")
		return r, err
	}
	return r, nil
}

func (c *SnapshotClient) GetSnapshotStatusAtBlockHeight(
	contract Contract,
	blockHeight uint64,
) (*SnapshotResponse, error) {
	if c.bypass() {
		return &SnapshotResponse{
			Data: SnapshotData{
				Message:     "",
				Status:      "success",
				BlockHeight: 1000000,
			},
		}, nil
	}

	var r *SnapshotResponse = &SnapshotResponse{}

	url := c.setSnapshotUrl(contract, "status-at-blockheight"+fmt.Sprintf("%d", blockHeight))
	req, err := c.setRequestMethod("GET", url, nil)
	if err != nil {
		log.Debug().Err(err).Msg("SnapshotClient GetSnapshotStatus request error")
		return r, err
	}

	if _, err := c.sendRequest(req, &r); err != nil {
		log.Debug().Err(err).Msg("SnapshotClient GetSnapshotStatus send request error")
		return r, err
	}

	return r, nil
}

func (c *SnapshotClient) GetAddressBalanceAtBlockHeight(
	address string,
	blockheight uint64,
	balanceResponse *FTBalanceResponse,
	contract *Contract,
) error {
	if c.bypass() {
		log.Info().Msgf("overriding snapshotter service for dev, getting local balance")
		balance, err := c.Fa.GetFlowBalance(address)
		if err != nil {
			log.Debug().Err(err).Msg(
				"SnapshotClient GetAddressBalanceAtBlockHeight request error",
			)
			return err
		}
		uintBalance := FloatBalanceToUint(balance)
		balanceResponse.PrimaryAccountBalance = uintBalance

		return nil
	}

	var url string

	if *contract.Name == "FlowToken" {
		url = fmt.Sprintf(`%s/balance-at-blockheight/%s/%d`, c.BaseURL, address, blockheight)
	} else {
		url = fmt.Sprintf(
			`%s/balance-at-blockheight/%s/%d/%v/%v`,
			c.BaseURL,
			address,
			blockheight,
			*contract.Addr,
			*contract.Name,
		)
	}

	req, err := c.setRequestMethod("GET", url, nil)
	if err != nil {
		log.Debug().Err(err).Msg("SnapshotClient GetAddressBalanceAtBlockHeight request error")
		return err
	}

	if _, err := c.sendRequest(req, balanceResponse); err != nil {
		log.Debug().Err(err).Msgf("Snapshot GetAddressBalanceAtBlockHeight send request error.")
		return err
	}
	log.Info().Msgf("got balance from snapshotter: %v", balanceResponse)

	return nil
}

func (c *SnapshotClient) GetLatestSnapshot(contract Contract) (*Snapshot, error) {
	var snapshot Snapshot
	var url string

	if c.bypass() {
		return &DummySnapshot, nil
	}

	//@TODO repeating logic here, refactor
	if *contract.Name == "FlowToken" {
		url = fmt.Sprintf(`%s/latest-snapshot`, c.BaseURL)
	} else {
		url = fmt.Sprintf(`%s/latest-snapshot/%s, %s`, c.BaseURL, *contract.Addr, *contract.Name)
	}

	req, err := c.setRequestMethod("GET", url, nil)
	if err != nil {
		log.Debug().Err(err).Msg("SnapshotClient GetAddressBalanceAtBlockHeight request error")
		return &snapshot, err
	}

	if _, err := c.sendRequest(req, snapshot); err != nil {
		log.Debug().Err(err).Msgf("Snapshot GetAddressBalanceAtBlockHeight send request error.")
		return &snapshot, err
	}

	return &snapshot, nil
}

func (c *SnapshotClient) AddFungibleToken(addr, name, path string) error {
	if c.bypass() {
		return nil
	}

	url := fmt.Sprintf(`%s/add-fungible-token`, c.BaseURL)

	payload := FungibleTokenContract{
		ContractAddress:      addr,
		ContractName:         name,
		PublicCapabilityPath: path,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Debug().Err(err).Msg("SnapshotClient POST request payload error")
		return err
	}

	req, err := c.setRequestMethod("POST", url, bytes.NewBuffer(body))
	if err != nil {
		log.Debug().Err(err).Msg("SnapshotClient AddFungibleToken request error")
		return err
	}

	resBody := struct {
		Data string `json:"data"`
	}{}
	if status, err := c.sendRequest(req, &resBody); err != nil {
		// snapshotter returns 400 when token exists. Ignore since it is not technically an error
		if status != 400 {
			log.Debug().Err(err).Msgf("Snapshot AddFungibleToken send request error.")
			return err
		}
	}

	return nil
}

func (c *SnapshotClient) GetLatestFlowSnapshot() (*Snapshot, error) {
	var snapshot Snapshot

	// Send dummy data for tests
	if c.bypass() {
		return &DummySnapshot, nil
	}

	url := c.BaseURL + "/latest-blockheight"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	if _, err := c.sendRequest(req, &snapshot); err != nil {
		log.Debug().Err(err).Msg("SnapshotClient GetLatestBlockHeightSnapshot request error")
		return nil, err
	}

	return &snapshot, nil
}

func (c *SnapshotClient) setSnapshotUrl(contract Contract, route string) string {
	var url string
	if *contract.Name == "FlowToken" {
		url = fmt.Sprintf(`%s/%s`, c.BaseURL, route)
	} else {
		url = fmt.Sprintf(`%s/%s/%v/%v`, c.BaseURL, route, *contract.Addr, *contract.Name)
	}

	return url
}

func (c *SnapshotClient) setRequestMethod(method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		log.Debug().Err(err).Msg("SnapshotClient TakeSnapshot request error")
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	return req, nil
}

func (c *SnapshotClient) sendRequest(req *http.Request, pointer interface{}) (int, error) {
	res, err := c.HTTPClient.Do(req)
	if err != nil {
		log.Debug().Err(err).Msg("snapshot http client error")
		return 500, err
	}

	defer res.Body.Close()

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusBadRequest {
		log.Debug().Msgf("snapshot error in sendRequest")
		return res.StatusCode, fmt.Errorf("unknown snapshot error, status code: %d", res.StatusCode)
	}

	log.Info().Msgf("body: %+v", res.Body)

	if err = json.NewDecoder(res.Body).Decode(pointer); err != nil {
		log.Debug().Err(err).Msgf("snapshot response decode error")
		return 500, err
	}

	return 200, nil
}

// Don't hit snapshot service if ENV is TEST or DEV
func (c *SnapshotClient) bypass() bool {
	return c.Env == "TEST" || c.Env == "DEV"
}
