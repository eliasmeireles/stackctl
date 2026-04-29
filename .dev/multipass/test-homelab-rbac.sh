#!/usr/bin/env bash
# End-to-end validation of the homelab-rbac flow inside the Multipass VM.
#
# Verifies:
#   - stackctl vault apply -f homelab-rbac.yaml creates namespaces, SA,
#     service-account-token Secret and RoleBindings (validation included)
#   - stackctl kubeconfig from-sa generates a working kubeconfig that the
#     SA can use to access its bound namespaces
#   - stackctl vault revert -f homelab-rbac.yaml cleans everything up
#
# The script port-forwards to the in-cluster vault Service so it does not
# depend on any host-level ingress hostname being reachable.
#
# Run inside the VM:
#   bash /home/ubuntu/workdir/test-homelab-rbac.sh

set -uo pipefail

export PATH=$PATH:/snap/bin:/home/ubuntu/go/bin:/root/go/bin
export KUBECONFIG=/home/ubuntu/workdir/kube/config

WORKDIR=/home/ubuntu/workdir
MANIFEST="${WORKDIR}/homelab-rbac.yaml"
TMP_KUBECONFIG="${WORKDIR}/dev-user.kubeconfig"
PF_LOG=/tmp/vault-port-forward.log
VAULT_PF_PORT=18200

PASS=0
FAIL=0

result() {
  local name="$1" rc="$2"
  if [ "$rc" -eq 0 ]; then
    echo "  ✓ PASS: $name"
    PASS=$((PASS+1))
  else
    echo "  ✗ FAIL: $name"
    FAIL=$((FAIL+1))
  fi
}

cleanup() {
  if [ -n "${PF_PID:-}" ] && kill -0 "${PF_PID}" 2>/dev/null; then
    kill "${PF_PID}" 2>/dev/null || true
    wait "${PF_PID}" 2>/dev/null || true
  fi
}
trap cleanup EXIT

echo "=========================================="
echo " stackctl homelab-rbac end-to-end test"
echo " Started at: $(date)"
echo "=========================================="

if ! command -v stackctl >/dev/null 2>&1; then
  echo "✗ stackctl not in PATH; aborting." ; exit 1
fi
if [ ! -f "${MANIFEST}" ]; then
  echo "✗ Manifest not found at ${MANIFEST}; aborting." ; exit 1
fi
if [ ! -f /home/ubuntu/workdir/vault/keys/root-token ]; then
  echo "✗ Vault root token not found; aborting." ; exit 1
fi

echo
echo "[setup] Port-forwarding vault Service to localhost:${VAULT_PF_PORT}..."
kubectl -n vault port-forward svc/vault "${VAULT_PF_PORT}:8200" >"${PF_LOG}" 2>&1 &
PF_PID=$!
for i in $(seq 1 20); do
  if curl -sf "http://127.0.0.1:${VAULT_PF_PORT}/v1/sys/health" >/dev/null 2>&1 \
     || curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:${VAULT_PF_PORT}/v1/sys/health" 2>/dev/null | grep -qE '^(200|429|473|501|503)$'; then
    break
  fi
  sleep 0.5
done

export VAULT_ADDR="http://127.0.0.1:${VAULT_PF_PORT}"
export VAULT_TOKEN
VAULT_TOKEN=$(cat /home/ubuntu/workdir/vault/keys/root-token)
export VAULT_SKIP_VERIFY=true

if ! curl -s -o /dev/null -w '%{http_code}' "${VAULT_ADDR}/v1/sys/health" 2>/dev/null | grep -qE '^(200|429|473|501|503)$'; then
  echo "✗ Vault is not reachable on ${VAULT_ADDR}; aborting." ; exit 1
fi

# Make sure no leftover from a previous run will create false positives.
kubectl delete rolebinding -n homelab-dev     dev-user-edit --ignore-not-found >/dev/null 2>&1 || true
kubectl delete rolebinding -n homelab-staging dev-user-edit --ignore-not-found >/dev/null 2>&1 || true
kubectl delete secret      -n kube-system     dev-user-token --ignore-not-found >/dev/null 2>&1 || true
kubectl delete sa          -n kube-system     dev-user --ignore-not-found >/dev/null 2>&1 || true
kubectl delete ns homelab-dev     --ignore-not-found >/dev/null 2>&1 || true
kubectl delete ns homelab-staging --ignore-not-found >/dev/null 2>&1 || true

echo
echo "[1/6] Validation: invalid manifest must fail before touching the cluster..."
cat >/tmp/bad-rbac.yaml <<'YAML'
kubernetes:
  role_bindings:
    - name: ""
      namespace: ""
      role_ref: {name: ""}
      subjects: []
YAML
if stackctl vault apply -f /tmp/bad-rbac.yaml >/tmp/bad.out 2>&1; then
  result "bad manifest is rejected by Validate" 1
else
  if grep -q "kubernetes config has" /tmp/bad.out; then
    result "bad manifest is rejected by Validate" 0
  else
    result "bad manifest is rejected by Validate" 1
    cat /tmp/bad.out
  fi
fi

echo
echo "[2/6] Applying homelab-rbac manifest..."
stackctl vault apply -f "${MANIFEST}"
result "stackctl vault apply" $?

echo
echo "[3/6] Checking resources exist in cluster..."
kubectl get ns homelab-dev      >/dev/null 2>&1 && result "ns/homelab-dev"        0 || result "ns/homelab-dev"        1
kubectl get ns homelab-staging  >/dev/null 2>&1 && result "ns/homelab-staging"    0 || result "ns/homelab-staging"    1
kubectl -n kube-system get sa dev-user             >/dev/null 2>&1 && result "sa/dev-user (kube-system)" 0 || result "sa/dev-user (kube-system)" 1
kubectl -n kube-system get secret dev-user-token   >/dev/null 2>&1 && result "secret/dev-user-token"     0 || result "secret/dev-user-token"     1

SECRET_TYPE=$(kubectl -n kube-system get secret dev-user-token -o jsonpath='{.type}' 2>/dev/null)
[ "${SECRET_TYPE}" = "kubernetes.io/service-account-token" ] && result "secret type is service-account-token" 0 || result "secret type is service-account-token" 1

kubectl -n homelab-dev     get rolebinding dev-user-edit >/dev/null 2>&1 && result "rb/dev-user-edit (homelab-dev)"     0 || result "rb/dev-user-edit (homelab-dev)"     1
kubectl -n homelab-staging get rolebinding dev-user-edit >/dev/null 2>&1 && result "rb/dev-user-edit (homelab-staging)" 0 || result "rb/dev-user-edit (homelab-staging)" 1

echo
echo "[4/6] Generating kubeconfig from ServiceAccount..."
sleep 2  # let the controller populate the token
rm -f "${TMP_KUBECONFIG}"
stackctl kubeconfig from-sa \
  --sa dev-user --namespace kube-system \
  --secret dev-user-token \
  --cluster-name homelab \
  --context-name dev-user@homelab \
  --default-namespace homelab-dev \
  --output-file "${TMP_KUBECONFIG}"
result "stackctl kubeconfig from-sa" $?

[ -s "${TMP_KUBECONFIG}" ] && result "generated kubeconfig is non-empty" 0 || result "generated kubeconfig is non-empty" 1

echo
echo "[5/8] Validating dev-user kubeconfig grants edit in bound namespaces..."
KUBECONFIG="${TMP_KUBECONFIG}" kubectl auth can-i create deployments -n homelab-dev     >/dev/null 2>&1 && result "dev-user can create deployments in homelab-dev"     0 || result "dev-user can create deployments in homelab-dev"     1
KUBECONFIG="${TMP_KUBECONFIG}" kubectl auth can-i create deployments -n homelab-staging >/dev/null 2>&1 && result "dev-user can create deployments in homelab-staging" 0 || result "dev-user can create deployments in homelab-staging" 1
KUBECONFIG="${TMP_KUBECONFIG}" kubectl auth can-i list nodes >/dev/null 2>&1 \
  && result "dev-user CANNOT list nodes (cluster-scoped denied)" 1 \
  || result "dev-user CANNOT list nodes (cluster-scoped denied)" 0

echo
echo "[6/8] Manifest-driven kubeconfig apply/revert (kind: KubeconfigFromSA)..."
KUBECONFIG_MANIFEST=/tmp/kubeconfig-from-sa.yaml
APPLY_OUT=/tmp/kubeconfig-from-sa.kubeconfig
rm -f "${APPLY_OUT}"
cat >"${KUBECONFIG_MANIFEST}" <<YAML
apiVersion: stackctl/v1
kind: KubeconfigFromSA
spec:
  serviceAccount: dev-user
  namespace: kube-system
  secret: dev-user-token
  clusterName: homelab
  contextName: dev-user@homelab-manifest
  defaultNamespace: homelab-staging
  outputFile: ${APPLY_OUT}
YAML

stackctl kubeconfig apply -f "${KUBECONFIG_MANIFEST}"
result "stackctl kubeconfig apply -f (outputFile)" $?
[ -s "${APPLY_OUT}" ] && result "outputFile created and non-empty" 0 || result "outputFile created and non-empty" 1
KUBECONFIG="${APPLY_OUT}" kubectl auth can-i create deployments -n homelab-staging >/dev/null 2>&1 \
  && result "manifest-generated kubeconfig grants edit in homelab-staging" 0 \
  || result "manifest-generated kubeconfig grants edit in homelab-staging" 1

stackctl kubeconfig revert -f "${KUBECONFIG_MANIFEST}"
result "stackctl kubeconfig revert -f (outputFile)" $?
[ ! -e "${APPLY_OUT}" ] && result "outputFile deleted by revert" 0 || result "outputFile deleted by revert" 1

# Idempotent revert — second run must succeed without error.
stackctl kubeconfig revert -f "${KUBECONFIG_MANIFEST}"
result "revert is idempotent (file already absent)" $?

# Bad manifest must be rejected before any side effect.
cat >/tmp/bad-kubeconfig.yaml <<'YAML'
kind: KubeconfigFromSA
spec:
  namespace: kube-system
YAML
if stackctl kubeconfig apply -f /tmp/bad-kubeconfig.yaml >/tmp/bad-kc.out 2>&1; then
  result "kubeconfig apply rejects manifest with missing serviceAccount" 1
else
  grep -q "serviceAccount is required" /tmp/bad-kc.out \
    && result "kubeconfig apply rejects manifest with missing serviceAccount" 0 \
    || { result "kubeconfig apply rejects manifest with missing serviceAccount" 1 ; cat /tmp/bad-kc.out ; }
fi

# Unknown kind must be rejected with a clear message.
cat >/tmp/unknown-kind.yaml <<'YAML'
kind: TotallyMadeUp
spec:
  foo: bar
YAML
if stackctl kubeconfig apply -f /tmp/unknown-kind.yaml >/tmp/unknown-kc.out 2>&1; then
  result "kubeconfig apply rejects unknown kind" 1
else
  grep -q "unsupported kind" /tmp/unknown-kc.out \
    && result "kubeconfig apply rejects unknown kind" 0 \
    || { result "kubeconfig apply rejects unknown kind" 1 ; cat /tmp/unknown-kc.out ; }
fi

# Active-kubeconfig merge path: empty outputFile → merge into KUBECONFIG.
MERGE_KUBECONFIG=/tmp/active-kubeconfig.yaml
cp "${KUBECONFIG}" "${MERGE_KUBECONFIG}"
KUBECONFIG_MANIFEST_MERGE=/tmp/kubeconfig-from-sa-merge.yaml
cat >"${KUBECONFIG_MANIFEST_MERGE}" <<YAML
apiVersion: stackctl/v1
kind: KubeconfigFromSA
spec:
  serviceAccount: dev-user
  namespace: kube-system
  secret: dev-user-token
  clusterName: homelab
  contextName: dev-user@homelab-merge
  defaultNamespace: homelab-dev
YAML

KUBECONFIG="${MERGE_KUBECONFIG}" stackctl kubeconfig apply -f "${KUBECONFIG_MANIFEST_MERGE}"
result "kubeconfig apply -f merges into active kubeconfig when outputFile empty" $?
KUBECONFIG="${MERGE_KUBECONFIG}" kubectl config get-contexts dev-user@homelab-merge >/dev/null 2>&1 \
  && result "merged context dev-user@homelab-merge present" 0 \
  || result "merged context dev-user@homelab-merge present" 1

KUBECONFIG="${MERGE_KUBECONFIG}" stackctl kubeconfig revert -f "${KUBECONFIG_MANIFEST_MERGE}"
result "kubeconfig revert -f removes merged context" $?
KUBECONFIG="${MERGE_KUBECONFIG}" kubectl config get-contexts dev-user@homelab-merge >/dev/null 2>&1 \
  && result "merged context removed by revert" 1 \
  || result "merged context removed by revert" 0

echo
echo "[7/8] stackctl version emits build metadata..."
VERSION_OUT=$(stackctl version)
echo "${VERSION_OUT}" | grep -q "Version:"    && result "version output has Version field"    0 || result "version output has Version field"    1
echo "${VERSION_OUT}" | grep -q "Built:"      && result "version output has Built field"      0 || result "version output has Built field"      1
echo "${VERSION_OUT}" | grep -q "Commit:"     && result "version output has Commit field"     0 || result "version output has Commit field"     1
echo "${VERSION_OUT}" | grep -q "Go version:" && result "version output has Go version field" 0 || result "version output has Go version field" 1

# version --short prints just the version string (script-friendly).
SHORT_OUT=$(stackctl version --short)
[ -n "${SHORT_OUT}" ] && [ "$(echo "${SHORT_OUT}" | wc -l)" = "1" ] \
  && result "version --short prints a single-line version" 0 \
  || result "version --short prints a single-line version" 1

# kubeconfig list alias works.
stackctl kubeconfig list >/dev/null 2>&1 \
  && result "kubeconfig list alias works" 0 \
  || result "kubeconfig list alias works" 1

# Dry-run: bad manifest is rejected with exit 1.
cat >/tmp/bad-dry.yaml <<'YAML'
kubernetes:
  service_accounts:
    - name: ""
      namespace: ""
YAML
if stackctl vault apply -f /tmp/bad-dry.yaml --dry-run >/tmp/bad-dry.out 2>&1; then
  result "vault apply --dry-run rejects bad manifest" 1
else
  grep -qi "validation failed" /tmp/bad-dry.out \
    && result "vault apply --dry-run rejects bad manifest" 0 \
    || { result "vault apply --dry-run rejects bad manifest" 1 ; cat /tmp/bad-dry.out ; }
fi
rm -f /tmp/bad-dry.yaml /tmp/bad-dry.out

# Dry-run: valid manifest succeeds without contacting cluster (no real-cluster prereq).
stackctl vault apply -f "${MANIFEST}" --dry-run >/dev/null 2>&1 \
  && result "vault apply --dry-run accepts valid manifest" 0 \
  || result "vault apply --dry-run accepts valid manifest" 1

# kubeconfig apply --dry-run also validates.
cat >/tmp/dry-kc.yaml <<'YAML'
kind: KubeconfigFromSA
spec:
  serviceAccount: dev-user
YAML
stackctl kubeconfig apply -f /tmp/dry-kc.yaml --dry-run >/dev/null 2>&1 \
  && result "kubeconfig apply --dry-run accepts valid manifest" 0 \
  || result "kubeconfig apply --dry-run accepts valid manifest" 1
rm -f /tmp/dry-kc.yaml

echo
echo "[8/8] Reverting homelab-rbac manifest and verifying cleanup..."
stackctl vault revert -f "${MANIFEST}"
result "stackctl vault revert" $?

# Namespace deletion is async — give it a moment
sleep 3
kubectl -n homelab-dev     get rolebinding dev-user-edit >/dev/null 2>&1 \
  && result "rb/dev-user-edit removed (homelab-dev)" 1 \
  || result "rb/dev-user-edit removed (homelab-dev)" 0
kubectl -n homelab-staging get rolebinding dev-user-edit >/dev/null 2>&1 \
  && result "rb/dev-user-edit removed (homelab-staging)" 1 \
  || result "rb/dev-user-edit removed (homelab-staging)" 0
kubectl -n kube-system get secret dev-user-token >/dev/null 2>&1 \
  && result "secret/dev-user-token removed" 1 \
  || result "secret/dev-user-token removed" 0
kubectl -n kube-system get sa dev-user >/dev/null 2>&1 \
  && result "sa/dev-user removed" 1 \
  || result "sa/dev-user removed" 0

rm -f "${TMP_KUBECONFIG}" /tmp/bad-rbac.yaml /tmp/bad.out \
      "${KUBECONFIG_MANIFEST}" "${KUBECONFIG_MANIFEST_MERGE}" "${MERGE_KUBECONFIG}" \
      /tmp/bad-kubeconfig.yaml /tmp/bad-kc.out \
      /tmp/unknown-kind.yaml /tmp/unknown-kc.out

echo
echo "=========================================="
echo "Tests passed: ${PASS}"
echo "Tests failed: ${FAIL}"
echo "Total:        $((PASS+FAIL))"
echo "Finished at:  $(date)"
echo "=========================================="
[ "${FAIL}" -eq 0 ] && exit 0 || exit 1
