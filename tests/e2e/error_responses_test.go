//go:build e2e

package e2e_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
	apiclientv1 "github.com/pershin-daniil/ninja-chat-bank/tests/e2e/api/client/v1"
	apimanagerv1 "github.com/pershin-daniil/ninja-chat-bank/tests/e2e/api/manager/v1"
)

var _ = Describe("Error Responses", Ordered, func() {
	var (
		ctx    context.Context
		cancel context.CancelFunc

		apiClientV1  *apiclientv1.ClientWithResponses
		apiManagerV1 *apimanagerv1.ClientWithResponses
	)

	BeforeAll(func() {
		ctx, cancel = context.WithCancel(suiteCtx)

		apiClientV1, _ = newClientAPI(ctx, clientsPool.Get())
	})

	AfterAll(func() {
		cancel()
	})

	It("401 unauthorized", func() {
		// Arrange.
		invalidAuthorizator := func(_ context.Context, req *http.Request) error {
			req.Header.Set("Authorization", "Bearer invalid_token")
			return nil
		}
		apiInvalidAccessToken, err := apiclientv1.NewClientWithResponses(
			apiClientV1Endpoint,
			apiclientv1.WithRequestEditorFn(invalidAuthorizator),
		)
		Expect(err).ShouldNot(HaveOccurred())

		// Action.
		resp, err := apiInvalidAccessToken.PostSendMessageWithResponse(ctx,
			&apiclientv1.PostSendMessageParams{XRequestID: types.NewRequestID()},
			apiclientv1.SendMessageRequest{},
		)

		// Assert.
		Expect(err).ShouldNot(HaveOccurred())
		expectSendClientMsgRespCode(resp, http.StatusUnauthorized)
	})

	It("413 bad request, too long message body", func() {
		// Arrange.
		tooLongMsgBody := strings.Repeat("a", 100_000)

		// Action.
		resp, err := apiClientV1.PostSendMessageWithResponse(ctx,
			&apiclientv1.PostSendMessageParams{XRequestID: types.NewRequestID()},
			apiclientv1.PostSendMessageJSONRequestBody{MessageBody: tooLongMsgBody},
		)

		// Assert.
		Expect(err).ShouldNot(HaveOccurred())
		expectSendClientMsgRespCode(resp, http.StatusRequestEntityTooLarge)
	})

	It("400 bad request, message body is empty", func() {
		// Action.
		resp, err := apiClientV1.PostSendMessageWithResponse(ctx,
			&apiclientv1.PostSendMessageParams{XRequestID: types.NewRequestID()},
			apiclientv1.PostSendMessageJSONRequestBody{MessageBody: ""},
		)
		Expect(err).ShouldNot(HaveOccurred())

		// Assert.
		Expect(err).ShouldNot(HaveOccurred())
		expectSendClientMsgRespCode(resp, http.StatusBadRequest)
	})

	It("5001 No active problem in chat", func() {
		// Arrange.
		_, err := apiClientV1.PostSendMessageWithResponse(ctx,
			&apiclientv1.PostSendMessageParams{XRequestID: types.NewRequestID()},
			apiclientv1.PostSendMessageJSONRequestBody{MessageBody: "Hello, sir!"},
		)
		Expect(err).ShouldNot(HaveOccurred())

		_, err = apiManagerV1.PostFreeHandsWithResponse(ctx,
			&apimanagerv1.PostFreeHandsParams{XRequestID: types.NewRequestID()},
		)
		Expect(err).ShouldNot(HaveOccurred())

		var chatID types.ChatID
		Eventually(ctx, func(g Gomega) { // wait for chat has occurred
			var resp *apimanagerv1.PostGetChatsResponse
			resp, err = apiManagerV1.PostGetChatsWithResponse(ctx,
				&apimanagerv1.PostGetChatsParams{XRequestID: types.NewRequestID()},
			)
			g.Expect(err).ShouldNot(HaveOccurred())

			g.Expect(resp.JSON200).ShouldNot(BeNil())
			g.Expect(resp.JSON200.Data).ShouldNot(BeNil())
			g.Expect(resp.JSON200.Data.Chats).ShouldNot(BeNil())
			g.Expect(resp.JSON200.Data.Chats).ShouldNot(BeEmpty())
			chatID = resp.JSON200.Data.Chats[0].ChatId
		}).WithTimeout(time.Second * 3).WithPolling(time.Millisecond * 100).Should(Succeed())

		_, err = apiManagerV1.PostCloseChat(ctx,
			&apimanagerv1.PostCloseChatParams{XRequestID: types.NewRequestID()},
			apimanagerv1.PostCloseChatJSONRequestBody{ChatId: chatID},
		)
		Expect(err).ShouldNot(HaveOccurred())

		// Action.
		closeResp, err := apiManagerV1.PostCloseChatWithResponse(ctx,
			&apimanagerv1.PostCloseChatParams{XRequestID: types.NewRequestID()},
			apimanagerv1.PostCloseChatJSONRequestBody{ChatId: chatID},
		)

		// Assert.
		Expect(err).ShouldNot(HaveOccurred())
		expectCloseChatRespCode(closeResp, 5001)
	})
})

func expectSendClientMsgRespCode[TCode ~int](resp *apiclientv1.PostSendMessageResponse, code TCode) {
	printResponse(resp.Body)
	Expect(resp).ShouldNot(BeNil())
	Expect(resp.JSON200).ShouldNot(BeNil())
	Expect(resp.JSON200.Error).ShouldNot(BeNil())
	Expect(resp.JSON200.Error.Code).Should(BeNumerically("==", code))
}

func expectCloseChatRespCode[TCode ~int](resp *apimanagerv1.PostCloseChatResponse, code TCode) {
	printResponse(resp.Body)
	Expect(resp).ShouldNot(BeNil())
	Expect(resp.JSON200).ShouldNot(BeNil())
	Expect(resp.JSON200.Error).ShouldNot(BeNil())
	Expect(resp.JSON200.Error.Code).Should(BeNumerically("==", code))
}

// printResponse helps to investigate server response in console.
func printResponse(rawBody []byte) {
	b, err := json.MarshalIndent(json.RawMessage(rawBody), "", "\t")
	Expect(err).ShouldNot(HaveOccurred())
	GinkgoWriter.Println(string(b))
}
