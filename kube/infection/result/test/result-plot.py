import json
import pathlib
from matplotlib import pyplot as plt
from random import randint

def read_json(name: str):
    json_open = open(name, 'r')
    json_load = json.load(json_open)
    print(json_load)
    return json_load

def get_filenames():
    filenames = []
    p_temp = pathlib.Path('./').glob('*.json')
    for p in p_temp:
        filenames.append(p.name)
    return filenames

def plot():
    # データの定義(サンプルなのでテキトー)
    x = list(range(10))
    y = [randint(0,100) for _ in x]

    # グラフの描画
    plt.plot(x, y)
    plt.show()

def data_generator():
    data = {}
    filenames = get_filenames()
    for name in filenames:
        json_data = read_json(name)
        for step_data in json_data:
            time = step_data["time"]
            data[time] = {}



if __name__ == "__main__":
    plot()
    #data_generator()