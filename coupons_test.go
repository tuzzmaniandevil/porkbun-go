package porkbun

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCoupons_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    Coupons
		wantErr string
	}{
		{
			name: "object of coupons",
			data: `{"registration":{"code":"AWESOMENESS","max_per_user":1,"first_year_only":"yes","type":"amount","amount":1}}`,
			want: Coupons{"registration": {
				Code:          "AWESOMENESS",
				MaxPerUser:    1,
				FirstYearOnly: "yes",
				Type:          "amount",
				Amount:        1,
			}},
		},
		{
			// No coupons is sent as an empty array rather than an empty object.
			name: "empty array",
			data: `[]`,
			want: nil,
		},
		{
			name: "empty object",
			data: `{}`,
			want: Coupons{},
		},
		{
			name:    "populated array",
			data:    `[{"code":"X"}]`,
			wantErr: "expected an object or an empty array",
		},
		{
			// The underlying reason has to survive: reporting only "unexpected
			// type" here would hide which field was wrong.
			name:    "wrong field type inside a coupon",
			data:    `{"registration":{"max_per_user":"not-a-number"}}`,
			wantErr: "max_per_user",
		},
		{
			name:    "not an object or array",
			data:    `"nope"`,
			wantErr: "coupons",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Coupons
			err := json.Unmarshal([]byte(tt.data), &got)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// first_year_only is a yes/no enum in the specification, so it is typed as YesNo
// like every other such field.
func TestCoupon_FirstYearOnlyIsTyped(t *testing.T) {
	var coupon Coupon
	require.NoError(t, json.Unmarshal([]byte(
		`{"code":"AWESOMENESS","max_per_user":1,"first_year_only":"yes","type":"amount","amount":1.5}`), &coupon))

	assert.Equal(t, Yes, coupon.FirstYearOnly)
	assert.Equal(t, int64(1), coupon.MaxPerUser)
	assert.InDelta(t, 1.5, coupon.Amount, 0.0001)
}
