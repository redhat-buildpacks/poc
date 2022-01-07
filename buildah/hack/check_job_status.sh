while true; do
  if kubectl -n buildpack wait --timeout=-1s --for=condition=complete job/buildah-poc 2>/dev/null; then
    job_result=0
    break
  fi

  if kubectl -n buildpack wait --timeout=-1s --for=condition=failed job/buildah-poc 2>/dev/null; then
    job_result=1
    break
  fi

  sleep 3
done

if [[ $job_result -eq 1 ]]; then
    echo "Job failed!"
    exit 1
fi

echo "Job succeeded"
exit 0