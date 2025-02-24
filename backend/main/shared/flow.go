package shared

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/onflow/cadence"
	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/client"
	"google.golang.org/grpc"
)

type FlowAdapter struct {
	Config  FlowConfig
	Client  *client.Client
	Context context.Context
	CustomScriptsMap map[string]CustomScript
	URL     string
	Env     string
}

type FlowContract struct {
	Source  string            `json:"source,omitempty"`
	Aliases map[string]string `json:"aliases"`
}

type FlowConfig struct {
	Contracts map[string]FlowContract `json:"contracts"`
	Networks  map[string]string       `json:"networks"`
}

type Contract struct {
	Name           *string  `json:"name,omitempty"`
	Addr           *string  `json:"addr,omitempty"`
	Public_path    *string  `json:"publicPath,omitempty"`
	Threshold      *float64 `json:"threshold,omitempty,string"`
	MaxWeight      *float64 `json:"maxWeight,omitempty,string"`
	Float_event_id *uint64  `json:"floatEventId,omitempty,string"`
	Script         *string  `json:"script,omitempty"`
}

var (
	placeholderTokenName            = regexp.MustCompile(`"[^"\s]*TOKEN_NAME"`)
	placeholderTokenAddr            = regexp.MustCompile(`"[^"\s]*TOKEN_ADDRESS"`)
	placeholderFungibleTokenAddr    = regexp.MustCompile(`"[^"\s]*FUNGIBLE_TOKEN_ADDRESS"`)
	placeholderNonFungibleTokenAddr = regexp.MustCompile(`"[^"\s]*NON_FUNGIBLE_TOKEN_ADDRESS"`)
	placeholderMetadataViewsAddr    = regexp.MustCompile(`"[^"\s]*METADATA_VIEWS_ADDRESS"`)
	placeholderCollectionPublicPath = regexp.MustCompile(`"[^"\s]*COLLECTION_PUBLIC_PATH"`)
	placeholderTopshotAddr          = regexp.MustCompile(`"[^"\s]*TOPSHOT_ADDRESS"`)
)

func NewFlowClient(flowEnv string, customScriptsMap map[string]CustomScript) *FlowAdapter {
	adapter := FlowAdapter{}
	adapter.Context = context.Background()
	adapter.Env = flowEnv
	adapter.CustomScriptsMap = customScriptsMap
	path := "./flow.json"

	content, err := ioutil.ReadFile(path)

	if err != nil {
		log.Fatal().Msgf("Error when opening file: %+v.", err)
	}

	var config FlowConfig
	err = json.Unmarshal(content, &config)
	if err != nil {
		log.Fatal().Msgf("Error parsing flow.json: %+v.", err)
	}

	adapter.Config = config
	adapter.URL = config.Networks[adapter.Env]

	// Explicitly set when running test suite
	if flag.Lookup("test.v") != nil {
		adapter.URL = "127.0.0.1:3569"
	}
	log.Info().Msgf("FLOW URL: %s", adapter.URL)

	// create flow client
	FlowClient, err := client.New(adapter.URL, grpc.WithInsecure())
	if err != nil {
		log.Panic().Msgf("Failed to connect to %s.", adapter.URL)
	}
	adapter.Client = FlowClient
	return &adapter
}

func (fa *FlowAdapter) GetAccountAtBlockHeight(addr string, blockheight uint64) (*flow.Account, error) {
	hexAddr := flow.HexToAddress(addr)
	return fa.Client.GetAccountAtBlockHeight(fa.Context, hexAddr, blockheight)
}

func (fa *FlowAdapter) GetCurrentBlockHeight() (int, error) {
	block, err := fa.Client.GetLatestBlock(fa.Context, true)
	if err != nil {
		return 0, err
	}
	return int(block.Height), nil
}

func (fa *FlowAdapter) ValidateSignature(address, message string, sigs *[]CompositeSignature, messageType string) error {
	log.Debug().Msgf("ValidateSignature()\nAddress: %s\nMessage: %s\nSigs: %v.", address, message, *sigs)

	// Prepare Script Args
	flowAddress := flow.HexToAddress(address)
	cadenceAddress := cadence.NewAddress(flowAddress)

	// Pull out signature strings + keyIds for script
	var cadenceSigs []cadence.Value = make([]cadence.Value, len(*sigs))
	var cadenceKeyIds []cadence.Value = make([]cadence.Value, len(*sigs))
	for i, cSig := range *sigs {
		cadenceKeyIds[i] = cadence.NewInt(int(cSig.Key_id))
		cadenceSigs[i] = cadence.String(cSig.Signature)
	}

	var domainSeparationTag string
	if messageType == "TRANSACTION" {
		domainSeparationTag = "FLOW-V0.0-transaction"
	} else {
		domainSeparationTag = "FLOW-V0.0-user"
	}

	// Load script
	script, err := ioutil.ReadFile("./main/cadence/scripts/validate_signature.cdc")

	if err != nil {
		log.Error().Err(err).Msgf("Error reading cadence script file.")
		return err
	}

	// call the script to verify the signature on chain
	value, err := fa.Client.ExecuteScriptAtLatestBlock(
		fa.Context,
		script,
		[]cadence.Value{
			cadenceAddress,
			cadence.NewArray(cadenceKeyIds),
			cadence.NewArray(cadenceSigs),
			cadence.String(message),
			cadence.String(domainSeparationTag),
		},
	)

	log.Info().Msgf("Validate signature script returned: %v", value)

	if err != nil && strings.Contains(err.Error(), "ledger returns unsuccessful") {
		log.Error().Err(err).Msg("signature validation error")
		return errors.New("flow access node error, please cast your vote again")
	} else if err != nil {
		log.Error().Err(err).Msg("Signature validation error.")
		return err
	}

	if err != nil {
		return err
	}

	if value != cadence.NewBool(true) {
		return errors.New("invalid signature")
	}

	return nil

}

func (fa *FlowAdapter) EnforceTokenThreshold(scriptPath, creatorAddr string, c *Contract) (bool, error) {

	var balance float64
	flowAddress := flow.HexToAddress(creatorAddr)
	cadenceAddress := cadence.NewAddress(flowAddress)
	cadencePath := cadence.Path{Domain: "public", Identifier: *c.Public_path}

	script, err := ioutil.ReadFile(scriptPath)
	if err != nil {
		log.Error().Err(err).Msgf("Error reading cadence script file.")
		return false, err
	}

	var cadenceValue cadence.Value

	if scriptPath == "./main/cadence/scripts/get_nfts_ids.cdc" {
		isFungible := false
		script = fa.ReplaceContractPlaceholders(string(script[:]), c, isFungible)

		//call the non-fungible token script to verify balance
		cadenceValue, err = fa.Client.ExecuteScriptAtLatestBlock(
			fa.Context,
			script,
			[]cadence.Value{
				cadenceAddress,
			})
		if err != nil {
			log.Error().Err(err).Msg("Error executing Non-Fungible-Token script.")
			return false, err
		}
		value := CadenceValueToInterface(cadenceValue)

		nftIds := value.([]interface{})
		balance = float64(len(nftIds))

	} else {
		isFungible := true
		script = fa.ReplaceContractPlaceholders(string(script[:]), c, isFungible)

		//call the fungible-token script to verify balance
		cadenceValue, err = fa.Client.ExecuteScriptAtLatestBlock(
			fa.Context,
			script,
			[]cadence.Value{
				cadencePath,
				cadenceAddress,
			})
		if err != nil {
			log.Error().Err(err).Msg("Error executing Funigble-Token Script.")
			return false, err
		}

		value := CadenceValueToInterface(cadenceValue)
		balance, err = strconv.ParseFloat(value.(string), 64)
		if err != nil {
			log.Error().Err(err).Msg("Error converting cadence value to float.")
			return false, err
		}
	}

	//check if balance is greater than threshold
	if balance < *c.Threshold {
		return false, nil
	}

	return true, nil
}

//this function only gets called in local dev when snapshot is being overidden
func (fa *FlowAdapter) GetFlowBalance(address string) (float64, error) {
	flowAddress := flow.HexToAddress(address)
	cadenceAddress := cadence.NewAddress(flowAddress)

	script, err := ioutil.ReadFile("./main/cadence/scripts/get_balance.cdc")
	if err != nil {
		log.Error().Err(err).Msgf("Error reading cadence script file.")
		return 0, err
	}

	contractName := "FlowToken"
	publicPath := "flowTokenBalance"
	contracAddress := "0x0ae53cb6e3f42a79"

	dummyContract := Contract{
		Name:        &contractName,
		Public_path: &publicPath,
		Addr:        &contracAddress,
	}

	script = fa.ReplaceContractPlaceholders(string(script[:]), &dummyContract, true)
	cadencePath := cadence.Path{Domain: "public", Identifier: *dummyContract.Public_path}

	cadenceValue, err := fa.Client.ExecuteScriptAtLatestBlock(
		fa.Context,
		script,
		[]cadence.Value{
			cadencePath,
			cadenceAddress,
		})
	if err != nil {
		log.Error().Err(err).Msg("Error executing Funigble-Token Script.")
		return 0, err
	}

	value := CadenceValueToInterface(cadenceValue)
	balance, err := strconv.ParseFloat(value.(string), 64)
	if err != nil {
		log.Error().Err(err).Msg("Error converting cadence value to float.")
		return 0, err
	}

	return balance, nil
}

func (fa *FlowAdapter) GetNFTIds(voterAddr string, c *Contract, path string) ([]interface{}, error) {
	flowAddress := flow.HexToAddress(voterAddr)
	cadenceAddress := cadence.NewAddress(flowAddress)

	script, err := ioutil.ReadFile(path)
	if err != nil {
		log.Error().Err(err).Msgf("Error reading cadence script file.")
		return nil, err
	}

	script = fa.ReplaceContractPlaceholders(string(script[:]), c, false)

	cadenceValue, err := fa.Client.ExecuteScriptAtLatestBlock(
		fa.Context,
		script,
		[]cadence.Value{
			cadenceAddress,
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("Error executing script.")
		return nil, err
	}

	value := CadenceValueToInterface(cadenceValue)

	nftIds := value.([]interface{})
	return nftIds, nil
}

func (fa *FlowAdapter) GetFloatNFTIds(voterAddr string, c *Contract) ([]interface{}, error) {
	flowAddress := flow.HexToAddress(voterAddr)
	cadenceAddress := cadence.NewAddress(flowAddress)
	cadenceUInt64 := cadence.NewUInt64(*c.Float_event_id)

	script, err := ioutil.ReadFile("./main/cadence/float/scripts/get_float_ids.cdc")
	if err != nil {
		log.Error().Err(err).Msgf("Error reading cadence script file.")
		return nil, err
	}

	script = fa.ReplaceContractPlaceholders(string(script[:]), c, false)

	cadenceValue, err := fa.Client.ExecuteScriptAtLatestBlock(
		fa.Context,
		script,
		[]cadence.Value{
			cadenceAddress,
			cadenceUInt64,
		})
	if err != nil {
		log.Error().Err(err).Msg("Error executing script.")
		return nil, err
	}

	value := CadenceValueToInterface(cadenceValue)

	nftIds := value.([]interface{})
	return nftIds, nil
}

func (fa *FlowAdapter) CheckIfUserHasEvent(voterAddr string, c *Contract) (bool, error) {
	flowAddress := flow.HexToAddress(voterAddr)
	cadenceAddress := cadence.NewAddress(flowAddress)
	cadenceUInt64 := cadence.NewUInt64(*c.Float_event_id)

	script, err := ioutil.ReadFile("./main/cadence/float/scripts/owns_specific_float.cdc")
	if err != nil {
		log.Error().Err(err).Msgf("Error reading cadence script file.")
		return false, err
	}

	script = fa.ReplaceContractPlaceholders(string(script[:]), c, false)

	cadenceValue, err := fa.Client.ExecuteScriptAtLatestBlock(
		fa.Context,
		script,
		[]cadence.Value{
			cadenceAddress,
			cadenceUInt64,
		})
	if err != nil {
		log.Error().Err(err).Msg("Error executing script.")
		return false, err
	}

	value := CadenceValueToInterface(cadenceValue)

	hasEventNFT := false
	if value == "true" {
		hasEventNFT = true
	}
	return hasEventNFT, nil
}

func (fa *FlowAdapter) GetEventNFT(voterAddr string, c *Contract) (interface{}, error) {

	//first we get all the floats a user owns

	flowAddress := flow.HexToAddress(voterAddr)
	cadenceAddress := cadence.NewAddress(flowAddress)
	cadenceUInt64 := cadence.NewUInt64(*c.Float_event_id)

	script, err := ioutil.ReadFile("./main/cadence/float/scripts/get_specific_float.cdc")
	if err != nil {
		log.Error().Err(err).Msgf("Error reading cadence script file.")
		return nil, err
	}

	script = fa.ReplaceContractPlaceholders(string(script[:]), c, false)

	cadenceValue, err := fa.Client.ExecuteScriptAtLatestBlock(
		fa.Context,
		script,
		[]cadence.Value{
			cadenceAddress,
			cadenceUInt64,
		})
	if err != nil {
		log.Error().Err(err).Msg("Error executing script.")
		return nil, err
	}

	value := CadenceValueToInterface(cadenceValue)

	return value, nil
}

func (fa *FlowAdapter) ReplaceContractPlaceholders(code string, c *Contract, isFungible bool) []byte {
	var (
		fungibleTokenAddr    string
		nonFungibleTokenAddr string
		metadataViewsAddr    string
		topshotAddr          string
	)

	nonFungibleTokenAddr = fa.Config.Contracts["NonFungibleToken"].Aliases[os.Getenv("FLOW_ENV")]
	fungibleTokenAddr = fa.Config.Contracts["FungibleToken"].Aliases[os.Getenv("FLOW_ENV")]
	metadataViewsAddr = fa.Config.Contracts["MetadataViews"].Aliases[os.Getenv("FLOW_ENV")]
	topshotAddr = fa.Config.Contracts["TopShot"].Aliases[os.Getenv("FLOW_ENV")]

	if isFungible {
		code = placeholderFungibleTokenAddr.ReplaceAllString(code, fungibleTokenAddr)
	} else {
		code = placeholderCollectionPublicPath.ReplaceAllString(code, *c.Public_path)
		code = placeholderNonFungibleTokenAddr.ReplaceAllString(code, nonFungibleTokenAddr)
	}

	code = placeholderMetadataViewsAddr.ReplaceAllString(code, metadataViewsAddr)
	code = placeholderTokenName.ReplaceAllString(code, *c.Name)
	code = placeholderTokenAddr.ReplaceAllString(code, *c.Addr)
	code = placeholderTopshotAddr.ReplaceAllString(code, topshotAddr)

	return []byte(code)
}

func WaitForSeal(
	ctx context.Context,
	c *client.Client,
	id flow.Identifier,
) (*flow.TransactionResult, *flow.Transaction, error) {
	result, err := c.GetTransactionResult(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	for result.Status != flow.TransactionStatusSealed {
		time.Sleep(time.Second)
		result, err = c.GetTransactionResult(ctx, id)
		if err != nil {
			return nil, nil, err
		}
	}

	tx, errTx := c.GetTransaction(ctx, id)
	return result, tx, errTx
}

func CadenceValueToInterface(field cadence.Value) interface{} {
	if field == nil {
		return ""
	}

	switch field := field.(type) {
	case cadence.Optional:
		return CadenceValueToInterface(field.Value)
	case cadence.Dictionary:
		result := map[string]interface{}{}
		for _, item := range field.Pairs {
			key, err := strconv.Unquote(item.Key.String())
			if err != nil {
				result[item.Key.String()] = CadenceValueToInterface(item.Value)
				continue
			}

			result[key] = CadenceValueToInterface(item.Value)
		}
		return result
	case cadence.Struct:
		result := map[string]interface{}{}
		subStructNames := field.StructType.Fields

		for j, subField := range field.Fields {
			result[subStructNames[j].Identifier] = CadenceValueToInterface(subField)
		}
		return result
	case cadence.Array:
		var result []interface{}
		for _, item := range field.Values {
			result = append(result, CadenceValueToInterface(item))
		}
		return result
	default:
		result, err := strconv.Unquote(field.String())
		if err != nil {
			return field.String()
		}
		return result
	}
}

func FloatBalanceToUint(balance float64) uint64 {
	return uint64(balance * 1000000)
}
