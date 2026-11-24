// Package signoz holds SigNoz's steps that no kind holds, which ws ubuntu guest signoz runs for
// the role's units. They are the root user's password, the wait for SigNoz's API, and its
// retention and dashboards, set through that API. mise copies foundryctl's compose files, and the
// kind compose ups the project.
package signoz

// Dir is the compose project's directory, which holds the compose files and root.env.
const Dir = "/opt/signoz"
