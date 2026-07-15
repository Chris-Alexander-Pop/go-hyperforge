// Package recaptcha implements captcha.Verifier via Google reCAPTCHA siteverify.
//
// This is a thin HTTP adapter (v2 checkbox / v3 score). It does not embed the
// client widget or manage site keys beyond Config.
package recaptcha
