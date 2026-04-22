//go:build darwin

package session

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework LocalAuthentication -framework Security -framework Foundation

#import <Foundation/Foundation.h>
#import <LocalAuthentication/LocalAuthentication.h>
#import <dispatch/dispatch.h>

int touch_id_available() {
	LAContext *context = [[LAContext alloc] init];
	if (context == nil) {
		return 0;
	}
	NSError *error = nil;
	BOOL canEvaluate = [context canEvaluatePolicy:LAPolicyDeviceOwnerAuthenticationWithBiometrics error:&error];
	[context release];
	return canEvaluate ? 1 : 0;
}

int touch_id_authenticate(char *reason) {
	LAContext *context = [[LAContext alloc] init];
	if (context == nil) {
		return -1;
	}
	NSError *error = nil;
	BOOL canEval = [context canEvaluatePolicy:LAPolicyDeviceOwnerAuthenticationWithBiometrics error:&error];
	if (!canEval) {
		[context release];
		return -2;
	}

	__block int result = 0;
	dispatch_semaphore_t semaphore = dispatch_semaphore_create(0);
	[context evaluatePolicy:LAPolicyDeviceOwnerAuthenticationWithBiometrics
	        localizedReason:[NSString stringWithUTF8String:reason]
	                  reply:^(BOOL success, NSError *replyError) {
		result = success ? 1 : 0;
		dispatch_semaphore_signal(semaphore);
	}];
	dispatch_semaphore_wait(semaphore, DISPATCH_TIME_FOREVER);
	[semaphore release];
	[context release];
	return result;
}
*/
import "C"

import (
	"context"
	"errors"
	"unsafe"
)

var errTouchIDNotAvailable = errors.New("touch id not available")
var errTouchIDFailed = errors.New("touch id authentication failed")

func touchIDAvailable() bool {
	return C.touch_id_available() == 1
}

func touchIDAuthenticate(ctx context.Context, reason string) error {
	if !touchIDAvailable() {
		return errTouchIDNotAvailable
	}
	cReason := C.CString(reason)
	defer C.free(unsafe.Pointer(cReason))
	result := C.touch_id_authenticate(cReason)
	if result == 1 {
		return nil
	}
	return errTouchIDFailed
}

type touchIDAuthenticator struct{}

func (t *touchIDAuthenticator) Authenticate(ctx context.Context, reason string) error {
	return touchIDAuthenticate(ctx, reason)
}

func (t *touchIDAuthenticator) IsAvailable() bool {
	return touchIDAvailable()
}

func newTouchIDAuthenticator() BiometricAuthenticator {
	return &touchIDAuthenticator{}
}

func init() {
	biometricAuthenticator = newTouchIDAuthenticator()
}
