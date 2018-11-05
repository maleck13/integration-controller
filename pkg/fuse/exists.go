package fuse

import (
	"github.com/sirupsen/logrus"
	v1alpha12 "github.com/syndesisio/syndesis/install/operator/pkg/apis/syndesis/v1alpha1"
)

// FuseExistsChecker checks fuse online exists
type FuseExistsChecker struct {
	ns         string
	fuseCruder Crudler
}

func NewFuseExistsChecker(ns string, crudler Crudler) *FuseExistsChecker {
	return &FuseExistsChecker{
		fuseCruder: crudler,
		ns:         ns,
	}
}

func (fe *FuseExistsChecker) Exists() bool {
	logrus.Debug("fuse consume: checking if a fuse exists")
	syndesisList := v1alpha12.NewSyndesisList()
	if err := fe.fuseCruder.List(fe.ns, syndesisList); err != nil {
		logrus.Error("fuse consumer: failed to check if fuse exists. Will try again ", err)
		return false
	}

	return len(syndesisList.Items) > 0
}
