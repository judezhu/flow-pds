package app

import (
	"fmt"

	"github.com/onflow/flow-go-sdk"
)

func (dist Distribution) Validate() error {
	if !dist.FlowID.Valid {
		return fmt.Errorf("distribution flowID must be defined")
	}

	if flow.Address(dist.Issuer) == flow.EmptyAddress {
		return fmt.Errorf("distribution issuer must be defined")
	}

	if err := dist.PackTemplate.Validate(); err != nil {
		return fmt.Errorf("error while validating pack template: %w", err)
	}

	return nil
}

func (pt PackTemplate) Validate() error {
	if pt.PackCount == 0 {
		return fmt.Errorf("pack count can not be zero")
	}

	if len(pt.Buckets) == 0 {
		return fmt.Errorf("no slot templates provided")
	}

	if err := pt.PackReference.Validate(); err != nil {
		return fmt.Errorf("error while validating PackReference: %w", err)
	}

	for i, bucket := range pt.Buckets {
		if err := bucket.Validate(); err != nil {
			return fmt.Errorf("error in slot template %d: %w", i+1, err)
		}

		requiredCount := int(pt.PackCount * bucket.CollectibleCount)
		allocatedCount := len(bucket.CollectibleCollection)
		if requiredCount > allocatedCount {
			return fmt.Errorf(
				"collection too small for slot template %d, required %d got %d",
				i+1, requiredCount, allocatedCount,
			)
		}
	}

	return nil
}

func (bucket Bucket) Validate() error {
	if bucket.CollectibleCount == 0 {
		return fmt.Errorf("collectible count can not be zero")
	}

	if err := bucket.CollectibleReference.Validate(); err != nil {
		return fmt.Errorf("error while validating CollectibleReference: %w", err)
	}

	if len(bucket.CollectibleCollection) == 0 {
		return fmt.Errorf("empty collection")
	}

	if int(bucket.CollectibleCount) > len(bucket.CollectibleCollection) {
		return fmt.Errorf(
			"collection too small, required %d got %d",
			int(bucket.CollectibleCount), len(bucket.CollectibleCollection),
		)
	}

	return nil
}

func (p Pack) Validate() error {
	if len(p.Collectibles) == 0 {
		return fmt.Errorf("no slots")
	}

	for i, c := range p.Collectibles {
		err := c.Validate()
		if err != nil {
			return fmt.Errorf("error while validating collectible in slot #%d: %w", i+1, err)
		}
	}

	return nil
}

func (al AddressLocation) Validate() error {
	if al.Name == "" {
		return fmt.Errorf("empty name")
	}
	if flow.Address(al.Address) == flow.EmptyAddress {
		return fmt.Errorf("empty address")
	}
	return nil
}

func (c Collectible) Validate() error {
	if !c.FlowID.Valid {
		return fmt.Errorf("collectible FlowID is not set")
	}
	if err := c.ContractReference.Validate(); err != nil {
		return fmt.Errorf("error while validating ContractReference: %w", err)
	}

	return nil
}
