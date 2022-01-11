while true; do
  if kubectl -n buildpack wait --timeout=300s --for=condition=complete job/kaniko-poc 2>/dev/null; then
    job_result=0
    break
  fi

  if kubectl -n buildpack wait --timeout=300s --for=condition=failed job/kaniko-poc 2>/dev/null; then
    job_result=1
    break
  fi

  kubectl -n buildpack get job/kaniko-poc -o yaml

  sleep 3
done

if [[ $job_result -eq 1 ]]; then
    echo "Job failed!"
    exit 1
fi

echo "Job succeeded"
exit 0