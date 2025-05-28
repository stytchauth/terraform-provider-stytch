package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

type Resp struct {
	Diagnostics diag.Diagnostics
}

func testAddErrorWithImmediateReturn(resp Resp, req any) {
	// ok: adderror-must-be-followed-by-return
	resp.Diagnostics.AddError(
		"Unexpected Data Source Configure Type",
		fmt.Sprintf("Expected *api.API, got: %T", req),
	)
	return
}

func testAddErrorWithImmediateReturnAfterBlankLine(resp Resp, req any) {
	// ok: adderror-must-be-followed-by-return
	resp.Diagnostics.AddError(
		"Unexpected Data Source Configure Type",
		fmt.Sprintf("Expected *api.API, got: %T", req),
	)

	return
}

func testAddErrorWithoutReturn(resp Resp, req any) {
	// ruleid: adderror-must-be-followed-by-return
	resp.Diagnostics.AddError(
		"Unexpected Data Source Configure Type",
		fmt.Sprintf("Expected *api.API, got: %T", req),
	)

	fmt.Println("error added")
	return
}

func testAddErrorWithReturnInline(resp Resp, req any) {
	if true {
		// ok: adderror-must-be-followed-by-return
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *api.API, got: %T", req))
		return
	}
}

func testAddErrorNoReturnAtAll(resp Resp, req any) {
	if true {
		// ruleid: adderror-must-be-followed-by-return
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *api.API, got: %T", req))
		fmt.Println("not returning")
	}
}

func testAddErrorNotFollowedByReturnInline(resp Resp, req any) {
	if true {
		// ruleid: adderror-must-be-followed-by-return
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *api.API, got: %T", req),
		)
		fmt.Println("returning!")
		return
	}
}

func ValidateConfig(resp Resp, req any) {
	// ok: adderror-must-be-followed-by-return
	resp.Diagnostics.AddError(
		"Unexpected Data Source Configure Type",
		fmt.Sprintf("Expected *api.API, got: %T", req),
	)
	resp.Diagnostics.AddError(
		"Something else",
		"Oh no!",
	)
}
