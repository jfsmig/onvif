// Copyright (C) 2022-2026 Jean-Francois SMIGIELSKI
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package utils

import (
	"errors"
	"fmt"
	"testing"
)

// The sentinels are wrapped at their call sites — networking.ReadAndParse attaches the
// HTTP status — so the contract that matters is that errors.Is still matches through the
// wrapping, not that the caller can compare with ==.
func TestSentinelSurvivesWrapping(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
	}{
		{"ErrHTTP", ErrHTTP},
		{"ErrNotOnvif", ErrNotOnvif},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wrapped := fmt.Errorf("%w: 401 Unauthorized", tc.err)
			if !errors.Is(wrapped, tc.err) {
				t.Fatalf("errors.Is lost the sentinel through one wrap: %v", wrapped)
			}
			if !errors.Is(fmt.Errorf("outer: %w", wrapped), tc.err) {
				t.Fatal("errors.Is lost the sentinel through two wraps")
			}
		})
	}
}

// The two sentinels must stay distinguishable: a constError compares by its string, so
// giving two of them the same text would silently make errors.Is match both.
func TestSentinelsAreDistinct(t *testing.T) {
	if errors.Is(ErrHTTP, ErrNotOnvif) {
		t.Fatal("ErrHTTP and ErrNotOnvif are indistinguishable")
	}
}
