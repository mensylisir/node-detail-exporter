#!/bin/bash

# 定义 fio 测试命令
FIO_FSYNC_CMD="fio --name=fsync --ioengine=sync --rw=write --bs=4k --numjobs=1 --filename=/var/lib/etcd/fio_test_fsync --runtime=60 --time_based --fsync=1 --size=1G --output-format=json"

# 定义输出文件
OUTPUT_FILE="configs/benchmarks.json"

echo "Running fio fsync benchmark..."
# 运行 fio 并将 JSON 输出重定向到临时文件
$FIO_FSYNC_CMD > /tmp/fio_fsync_result.json

# 从 fio 的 JSON 输出中提取我们需要的值
# 我们关心 sync percentiles (fsync 延迟)
# fio 的 JSON 输出比较复杂，这里用 jq 进行解析
FSYNC_P99=$(jq -r '.jobs[0].sync.percentile."99.000000"' /tmp/fio_fsync_result.json)
FSYNC_AVG=$(jq -r '.jobs[0].sync.mean' /tmp/fio_fsync_result.json)

# 将延迟从微秒 (usec) 转换为秒，以符合 Prometheus 的单位标准
FSYNC_P99_SEC=$(echo "$FSYNC_P99 / 1000000" | bc -l)
FSYNC_AVG_SEC=$(echo "$FSYNC_AVG / 1000000" | bc -l)

echo "Generating benchmark file at ${OUTPUT_FILE}..."
# 创建最终的 JSON 文件
cat <<EOF > ${OUTPUT_FILE}
{
"fsync_p99_seconds": ${FSYNC_P99_SEC},
"fsync_avg_seconds": ${FSYNC_AVG_SEC}
}
EOF

echo "Benchmark file created successfully:"
cat $OUTPUT_FILE

# 清理临时文件
rm /tmp/fio_fsync_result.json
