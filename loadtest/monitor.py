#!/usr/bin/env python3
"""monitor.py writes a line of CSV every few seconds: how busy the machine is,
and how much CPU and memory each part of the app uses.

    python3 monitor.py --out monitor.csv --part nginx=nginx.service \\
        --part api=docker-<id>.scope --part postgres=docker-<id>.scope

CPU is in cores: 1.0 means one core fully busy. It reads Linux's own counters
(cgroup v2 and /proc), so it costs almost nothing and doesn't disturb the test.
Stop it with Ctrl+C or kill.
"""

import argparse
import os
import subprocess
import time

CGROUPS = '/sys/fs/cgroup/system.slice'


def machine_cpu():
    # The first line of /proc/stat: time spent in each state, in clock ticks.
    with open('/proc/stat') as f:
        fields = [int(v) for v in f.readline().split()[1:]]
    idle = fields[3] + fields[4]  # idle + iowait
    return sum(fields), idle, fields[4]


def part_cpu_usec(scope):
    with open(f'{CGROUPS}/{scope}/cpu.stat') as f:
        for line in f:
            key, value = line.split()
            if key == 'usage_usec':
                return int(value)
    return 0


def part_memory_mb(scope):
    with open(f'{CGROUPS}/{scope}/memory.current') as f:
        return int(f.read()) / 1e6


def memory_used_mb():
    info = {}
    with open('/proc/meminfo') as f:
        for line in f:
            key, value = line.split(':')
            info[key] = int(value.split()[0])
    return (info['MemTotal'] - info['MemAvailable']) / 1e3


def network_bytes(interface):
    with open('/proc/net/dev') as f:
        for line in f:
            if line.strip().startswith(interface + ':'):
                fields = line.split(':')[1].split()
                return int(fields[0]), int(fields[8])
    return 0, 0


def established(port):
    out = subprocess.run(['ss', '-Htn', 'state', 'established', f'( sport = :{port} )'],
                         capture_output=True, text=True).stdout
    return len(out.splitlines())


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--out', required=True)
    parser.add_argument('--interval', type=float, default=5)
    parser.add_argument('--interface', default='ens5')
    parser.add_argument('--port', type=int, default=443, help='count connections to this port')
    parser.add_argument('--part', action='append', default=[], help='name=cgroup scope')
    args = parser.parse_args()
    parts = [p.split('=', 1) for p in args.part]
    cores = os.cpu_count()

    header = ['time', 'cpu_cores_used', 'cpu_iowait_cores', 'load1', 'mem_used_mb',
              'net_rx_mbit', 'net_tx_mbit', f'conns_{args.port}']
    for name, _ in parts:
        header += [f'{name}_cores', f'{name}_mem_mb']

    prev_total, prev_idle, prev_wait = machine_cpu()
    prev_rx, prev_tx = network_bytes(args.interface)
    prev_parts = [part_cpu_usec(scope) for _, scope in parts]
    prev_time = time.monotonic()

    with open(args.out, 'w', buffering=1) as out:
        out.write(','.join(header) + '\n')
        while True:
            time.sleep(args.interval)
            now = time.monotonic()
            elapsed = now - prev_time
            total, idle, wait = machine_cpu()
            busy = 1 - (idle - prev_idle) / max(1, total - prev_total)
            iowait = (wait - prev_wait) / max(1, total - prev_total)
            rx, tx = network_bytes(args.interface)
            with open('/proc/loadavg') as f:
                load1 = f.read().split()[0]
            row = [
                time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime()),
                f'{busy * cores:.2f}',
                f'{iowait * cores:.2f}',
                load1,
                f'{memory_used_mb():.0f}',
                f'{(rx - prev_rx) * 8 / elapsed / 1e6:.1f}',
                f'{(tx - prev_tx) * 8 / elapsed / 1e6:.1f}',
                str(established(args.port)),
            ]
            usages = []
            for (name, scope), before in zip(parts, prev_parts):
                usec = part_cpu_usec(scope)
                usages.append(usec)
                row += [f'{(usec - before) / 1e6 / elapsed:.2f}', f'{part_memory_mb(scope):.0f}']
            out.write(','.join(row) + '\n')
            prev_total, prev_idle, prev_wait = total, idle, wait
            prev_rx, prev_tx = rx, tx
            prev_parts = usages
            prev_time = now


if __name__ == '__main__':
    main()
