package responses

type SubsonicError int

//goland:noinspection GoUnusedConst
const (
	SubsonicErrorGeneric                                                  SubsonicError = 0
	SubsonicErrorRequiredParameterMissing                                 SubsonicError = 10
	SubsonicErrorIncompatibleSubsonicRestProtocolVersionClientMustUpgrade SubsonicError = 20
	SubsonicErrorIncompatibleSubsonicRestProtocolVersionServerMustUpgrade SubsonicError = 30
	SubsonicErrorWrongUsernameOrPassword                                  SubsonicError = 40
	SubsonicErrorTokenAuthenticationNotSupported                          SubsonicError = 41
	SubsonicErrorProvidedAuthenticationMechanismNotSupported              SubsonicError = 42
	SubsonicErrorMultipleConflictingAuthenticationMechanismsProvided      SubsonicError = 43
	SubsonicErrorInvalidAPIKey                                            SubsonicError = 44
	SubsonicErrorUserNotAuthorized                                        SubsonicError = 50
	SubsonicErrorTrialOver                                                SubsonicError = 60
	SubsonicErrorNotFound                                                 SubsonicError = 70
)
