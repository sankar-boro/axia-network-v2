// Copyright (C) 2019-2021, Axia Systems, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package rpc

import (
	"context"
	"fmt"
	"net/url"
)

var _ EndpointRequester = &axiaEndpointRequester{}

type EndpointRequester interface {
	SendRequest(ctx context.Context, method string, params interface{}, reply interface{}, options ...Option) error
}

type axiaEndpointRequester struct {
	uri, base string
}

func NewEndpointRequester(uri, base string) EndpointRequester {
	return &axiaEndpointRequester{
		uri:  uri,
		base: base,
	}
}

func (e *axiaEndpointRequester) SendRequest(
	ctx context.Context,
	method string,
	params interface{},
	reply interface{},
	options ...Option,
) error {
	uri, err := url.Parse(e.uri)
	if err != nil {
		return err
	}
	return SendJSONRequest(
		ctx,
		uri,
		fmt.Sprintf("%s.%s", e.base, method),
		params,
		reply,
		options...,
	)
}
