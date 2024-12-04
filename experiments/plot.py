#! /usr/bin/env python3

import matplotlib.pyplot as plt
import numpy as np
import os
import argparse
from parse import parse

def parseLog(filepath):
    print("Parsing", filepath)
    with open(filepath, 'r') as f:
        lines = f.readlines()
    throughput = None
    slowdown = None
    for i in range(len(lines)):
        lines[i] = lines[i].strip()
        if "Throughput" in lines[i]:
            throughput = parse("Throughput: {:d} req/s", lines[i])[0]
        if "Slowdown" in lines[i]:
            slowdown = parse("Slowdown: {} p99={}({p99:f})", lines[i])["p99"]
    return throughput, slowdown

def parseBaseline(name):
    latencys = []
    tputs = []
    for f in os.listdir("results"):
        if name not in f:
            continue
        tput, slow = parseLog(os.path.join("results", f))
        tputs.append(tput)
        latencys.append(slow)
    tputs.sort()
    latencys.sort()
    return tputs, latencys

def plot(kayak, pyxis, out):
    font_size = 20
    markersize = 10
    linewidth = 3
    plt.rc("font", **{"size": font_size})
    plt.figure(figsize=(10, 8))
    plt.ylabel("Throughput (requests/s)", labelpad=8)
    plt.xlabel("SLO", labelpad=8)
    plt.grid(True)

    ######################################## plot
    labels = ["Kayak", "Pyxis"]
    markers = ["o", "s"]
    colors = ["orange", "blue"]

    plt.plot(kayak[1],kayak[0],label="Kayak",marker="o",markersize=markersize,markeredgecolor="k",color="orange",linestyle="-",linewidth=linewidth)
    plt.plot(pyxis[1],pyxis[0],label="Pyxis",marker="s",markersize=markersize,markeredgecolor="k",color="blue",linestyle="-",linewidth=linewidth)
    # plt.xscale("log")

    ############################################ legend
    legend_size = 20
    legend_length = 2
    # bbox_to_anchor = (0.5, 1.0)
    plt.legend(
        # loc="upper center",
        # bbox_to_anchor=bbox_to_anchor,
        handlelength=legend_length,
        frameon=True,
        borderaxespad=0.5,
        prop={"size": legend_size},
    )
    os.makedirs("figures", exist_ok=True)
    plt.savefig(os.path.join("figures", out), bbox_inches="tight", transparent=False)
    

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("run", metavar="run", type=int, default=1)
    args = parser.parse_args()
    run = args.run

    baseDir = os.path.join(os.path.dirname(os.path.realpath(__file__)),f"{run}")
    os.chdir(baseDir)

    # Parse logs
    kayak = parseBaseline("kayak")
    pyxis = parseBaseline("pyxis")

    # Plot
    plot(kayak, pyxis, "tput-slo.png")