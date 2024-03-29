package internal

import (
	"reflect"
	"testing"
)

func TestNewGuessConfig(t *testing.T) {
	possibleGuesses := "1,2,3,4,5"
	possibleGuessesDesc := "A,B,C,D,E"
	want := &GuessConfig{
		Guesses: []guessConfigEntry{
			{
				Guess:       1,
				Description: "A",
			},
			{
				Guess:       2,
				Description: "B",
			},
			{
				Guess:       3,
				Description: "C",
			},
			{
				Guess:       4,
				Description: "D",
			},
			{
				Guess:       5,
				Description: "E",
			},
		},
	}

	got, err := NewGuessConfig(possibleGuesses, possibleGuessesDesc)

	if err != nil {
		t.Error("expected error to be nil")
	}

	if !reflect.DeepEqual(want, got) {
		t.Errorf("want %v, got %v", want, got)
	}
}

func TestNewGuessConfig_WhenLengthDiffers(t *testing.T) {
	tests := []struct {
		name                string
		possibleGuesses     string
		possibleGuessesDesc string
		expectedErr         string
	}{
		{
			name:                "guesses differ from description",
			possibleGuesses:     "1,2,3,4",
			possibleGuessesDesc: "A,B,C,D,E",
			expectedErr:         "error length for guesses and guesses desc is differrent (guesses = 4, guesses desc = 5)",
		},
		{
			name:                "description differ from guesses",
			possibleGuesses:     "1,2,3",
			possibleGuessesDesc: "A,B",
			expectedErr:         "error length for guesses and guesses desc is differrent (guesses = 3, guesses desc = 2)",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewGuessConfig(test.possibleGuesses, test.possibleGuessesDesc)

			if err.Error() != test.expectedErr {
				t.Errorf("expected error %v", test.expectedErr)
			}
		})
	}
}

func TestNewGuessConfig_WhenGuessIsNotNumeric(t *testing.T) {
	possibleGuesses := "1,2,P,4,5"
	possibleGuessesDesc := "A,B,C,D,E"

	expectedErr := "error can not convert guess P to int"

	_, err := NewGuessConfig(possibleGuesses, possibleGuessesDesc)
	if err.Error() != expectedErr {
		t.Errorf("expected error %v", expectedErr)
	}
}
