package lambdaproxy

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
)

var isbnRegexp = regexp.MustCompile(`[0-9]{3}\-[0-9]{10}`)
var errorLogger = log.New(os.Stderr, "ERROR ", log.Llongfile)

type apiGatewayProxyRequest struct {
	Resource              string            `json:"resource"` // The resource path defined in API Gateway
	Path                  string            `json:"path"`     // The url path for the caller
	HTTPMethod            string            `json:"httpMethod"`
	Headers               map[string]string `json:"headers"`
	QueryStringParameters map[string]string `json:"queryStringParameters"`
	Body                  string            `json:"body"`
	IsBase64Encoded       bool              `json:"isBase64Encoded,omitempty"`
}

// Add a helper for handling errors. This logs any error to os.Stderr
// and returns a 500 Internal Server Error response that the AWS API
// Gateway understands.
func serverError(err error) (apiGatewayProxyResponse, error) {
	errorLogger.Println(err.Error())

	return apiGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       http.StatusText(http.StatusInternalServerError),
	}, nil
}

// Similarly add a helper for send responses relating to client errors.
func clientError(status int) (apiGatewayProxyResponse, error) {
	return apiGatewayProxyResponse{
		StatusCode: status,
		Body:       http.StatusText(status),
	}, nil
}

type HTTPRequest struct {
	Method   string   `json:"method"`
	Resource string   `json:"resource"`
	Headers  []string `json:"headers"`
	Body     string   `json:"body"`
	Username string   `json:"username"`
	Password string   `json:"password"`
}
type HTTPResponse struct {
	StatusCode int      `json:"statusCode"`
	Headers    []string `json:"headers"`
	Body       string   `json:"body"`
}

func handleRequest(ctx context.Context, request *HTTPProbeCmd) (*HTTPRequest, error) {

	fmt.Printf("Body size = %d.\n", len(request.Body))

	fmt.Println("Headers:")
	for key, value := range request.Headers {
		fmt.Printf("    %s: %s\n", key, value)
	}

	return &HTTPRequest{Method: request.Method, Resource: request.Resource, Headers: request.Headers, Body: request.Body}, nil
}

func main() {
	input, err := handleRequest(context.Background(), &HTTPProbeCmd{})
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	EncodeRequest(input, nil)
}

func EncodeRequest(input *HTTPRequest, options *EncodeOptions) ([]byte, error) {

	// encode http request to api gateway proxy request
	// to do matching of parsing of both structs
	encodeapiGatewayStruct := apiGatewayProxyRequest{
		Resource: input.Resource,
		Body:     input.Body,
		Headers:  convertSliceToMap(input.Headers),
	}

	var b bytes.Buffer
	encoder := json.NewEncoder(&b)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(&encodeapiGatewayStruct); err != nil {
		fmt.Errorf("Error encoding apiGatewayProxyRequest: %s", err)
		return nil, err
	}

	return b.Bytes(), nil
}

func DecodeResponse(input []byte, options DecodeOptions) (HTTPResponse, error) {

	var response apiGatewayProxyResponse
	if err := json.NewDecoder(bytes.NewBuffer(input)).Decode(&response); err != nil {
		return HTTPResponse{}, err
	}
	//decode the response.Body if base64 encoded
	if response.IsBase64Encoded {
		responseBodyBytes, err := base64.StdEncoding.DecodeString(string(response.Body))
		if err != nil {
			log.Fatalf("Some error occured during base64 decode. Error %s", err.Error())
		}
		response.Body = string(responseBodyBytes)
	}

	return HTTPResponse{StatusCode: response.StatusCode, Body: response.Body, Headers: convertMapToSlice(response.Headers)}, nil
}

func convertSliceToMap(input []string) map[string]string {
	output := map[string]string{}
	for _, v := range input {
		output[v] = v
	}
	return output
}

func convertMapToSlice(input map[string]string) []string {
	output := []string{}
	for _, v := range input {
		output = append(output, v)
	}
	return output
}
