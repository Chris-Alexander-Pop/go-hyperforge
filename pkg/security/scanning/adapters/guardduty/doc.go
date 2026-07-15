/*
Package guardduty is a thin scanning.Scanner adapter over AWS GuardDuty List/GetFindings.

Inject FindingsAPI via NewFromAPI for unit tests; New wires the AWS SDK client.
*/
package guardduty
