/**
  Copyright (c) 2022 Arpabet, LLC. All rights reserved.
*/

package beans_test

import (
	"github.com/stretchr/testify/require"
	"go.arpabet.com/beans"
	"testing"
)

/**
Cycle dependency test of plain beans
*/

type aPlainBean struct {
	BBean *bPlainBean `inject`
}

type bPlainBean struct {
	CBean *cPlainBean `inject`
}

type cPlainBean struct {
	ABean *aPlainBean `inject:"lazy"`
}

func TestPlainBeanCycle(t *testing.T) {

	beans.Verbose = true

	ctx, err := beans.Create(
		&aPlainBean{},
		&bPlainBean{},
		&cPlainBean{},
	)
	require.NoError(t, err)
	defer ctx.Close()

}

type selfDepBean struct {
	Self *selfDepBean `inject`
}

func TestSelfDepCycle(t *testing.T) {

	beans.Verbose = true

	self := &selfDepBean{}

	ctx, err := beans.Create(
		self,
	)
	require.NoError(t, err)
	defer ctx.Close()

	require.True(t, self == self.Self)

}