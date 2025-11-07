#!/bin/bash

# Define the output file for the benchmark results
OUTPUT_FILE="configs/benchmarks.json"

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Initialize metrics to null
FSYNC_P99_SEC="null"
CPU_EVENTS_PER_SECOND="null"
MEM_OPS_PER_SECOND="null"
NET_BANDWIDTH_BITS_PER_SECOND="null"

# --- FIO Benchmark ---
if command_exists fio && command_exists jq && command_exists bc; then
    echo "Running fio fsync benchmark..."
    # Shortened runtime and size to prevent timeouts
    FIO_FSYNC_CMD="fio --name=fsync --ioengine=sync --rw=write --bs=4k --numjobs=1 --filename=fio_test_fsync --runtime=10 --time_based --fsync=1 --size=100M --output-format=json"
    $FIO_FSYNC_CMD > /tmp/fio_fsync_result.json
    if [ $? -eq 0 ]; then
        FSYNC_P99=$(jq -r '.jobs[0].sync.percentile."99.000000"' /tmp/fio_fsync_result.json)
        if [[ "$FSYNC_P99" != "null" && "$FSYNC_P99" =~ ^[0-9.]+$ ]]; then
            FSYNC_P99_SEC=$(echo "scale=6; $FSYNC_P99 / 1000000" | bc)
        fi
        rm /tmp/fio_fsync_result.json
    else
        echo "Warning: fio command failed. Skipping disk benchmark."
    fi
else
    echo "Warning: fio, jq, or bc is not installed. Skipping disk benchmark."
fi

# --- Sysbench CPU Benchmark ---
if command_exists sysbench; then
    echo "Running sysbench CPU benchmark..."
    CPU_THREADS=$(nproc)
    # Shortened runtime to prevent timeouts
    SYSBENCH_CPU_CMD="sysbench cpu --threads=${CPU_THREADS} --cpu-max-prime=10000 --time=10 run"
    CPU_EVENTS_PER_SECOND=$($SYSBENCH_CPU_CMD | grep 'events per second:' | awk '{print $4}')
    if [[ ! "$CPU_EVENTS_PER_SECOND" =~ ^[0-9.]+$ ]]; then
        CPU_EVENTS_PER_SECOND="null"
    fi
else
    echo "Warning: sysbench is not installed. Skipping CPU benchmark."
fi

# --- Sysbench Memory Benchmark ---
if command_exists sysbench; then
    echo "Running sysbench memory benchmark..."
    CPU_THREADS=$(nproc)
    # Shortened runtime and size to prevent timeouts
    SYSBENCH_MEM_CMD="sysbench memory --threads=${CPU_THREADS} --memory-block-size=1M --memory-total-size=1G --time=10 run"
    MEM_OPS_PER_SECOND=$($SYSBENCH_MEM_CMD | grep 'operations performed:' | awk '{print $5}' | tr -d '()')
     if [[ ! "$MEM_OPS_PER_SECOND" =~ ^[0-9.]+$ ]]; then
        MEM_OPS_PER_SECOND="null"
    fi
else
    echo "Warning: sysbench is not installed. Skipping memory benchmark."
fi

# --- iperf3 Network Benchmark ---
if command_exists iperf3 && command_exists jq; then
    echo "Running iperf3 network benchmark..."
    iperf3 -s &
    IPERF_SERVER_PID=$!
    sleep 1
    # Shortened runtime
    IPERF_CLIENT_OUTPUT=$(iperf3 -c 127.0.0.1 -t 5 -J)
    kill $IPERF_SERVER_PID > /dev/null 2>&1
    wait $IPERF_SERVER_PID 2>/dev/null

    if [ $? -eq 0 ]; then
        NET_BANDWIDTH_BITS_PER_SECOND=$(echo $IPERF_CLIENT_OUTPUT | jq '.end.sum_received.bits_per_second')
        if [[ "$NET_BANDWIDTH_BITS_PER_SECOND" == "null" ]]; then
            echo "Warning: Failed to parse iperf3 output."
        fi
    else
        NET_BANDWIDTH_BITS_PER_SECOND="null"
        echo "Warning: iperf3 command failed. Skipping network benchmark."
    fi
else
    echo "Warning: iperf3 or jq is not installed. Skipping network benchmark."
fi

# --- Generate JSON Output ---
echo "Generating benchmark file at ${OUTPUT_FILE}..."
cat <<EOF > ${OUTPUT_FILE}
{
  "fsync_p99_seconds": ${FSYNC_P99_SEC},
  "cpu_events_per_second": ${CPU_EVENTS_PER_SECOND},
  "memory_ops_per_second": ${MEM_OPS_PER_SECOND},
  "net_bandwidth_bits_per_second": ${NET_BANDWIDTH_BITS_PER_SECOND}
}
EOF

echo "Benchmark file created successfully:"
cat $OUTPUT_FILE
