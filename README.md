# Pyxis over K8s

This repository contains artifacts for running [Pyxis](https://ieeexplore.ieee.org/document/10570339) over K8s. It is ported from the original Pyxis [prototype](https://github.com/TomQuartz/Pyxis)

## Steps

1. Set up a K8s cluster.
2. Run `scripts/run.sh $RUN`. Replace `$RUN` with your custom experiment ID.
3. Find the visualized results at `experiments/$RUN/figures`

## License

All source code in this repository is licensed under Apache License 2.0.