let updateInterval;
const updateFrequency = 2.5 * 1000; // Frequency of updates in milliseconds
let chart
const chartOptions = {
    chart: {
        type: 'spline',
        animation: false,
        reflow: false,
        zoomType: 'x',
        panning: true,
        panKey: 'shift',
        height: 360,
    },
    title: {
        text: 'Производительность'
    },
    xAxis: {
        type: 'datetime',
        title: {
            text: 'Время'
        },
        dateTimeLabelFormats: {
            month: '%e. %b'
        },
        crosshair: true
    },
    yAxis: {
        title: {
            text: 'Количество'
        },
        lineWidth: 1
    },
    legend: {
        layout: 'vertical',
        align: 'right',
        verticalAlign: 'middle'
    },
    tooltip: {
        shared: true
    },
    plotOptions: {
        series: {
            marker: {
                radius: 1
            }
        }
    },
    series: [{
        name: 'Количество потоков',
        data: []
    },{
        name: "Длинна очереди",
        data: []
    }],
    responsive: {
        rules: [{
            condition: {
                maxWidth: 500
            },
            chartOptions: {
                legend: {
                    layout: 'horizontal',
                    align: 'center',
                    verticalAlign: 'bottom'
                }
            }
        }]
    }
};


function updateChart(data) {
    const dataArrayValues1 = Object.values(data).map(item =>(item.threadsChecker));
    const dataArrayValues2 = Object.values(data).map(item =>(item.lenQueue));
    const dataArrayKeys =  Object.keys(data).map(item => (item*1000))
    const seriesData1 = dataArrayKeys.map((item, index) => [item, dataArrayValues1[index]]);
    const seriesData2 = dataArrayKeys.map((item, index) => [item, dataArrayValues2[index]]);
    chart.series[0].setData(seriesData1);
    chart.series[1].setData(seriesData2);
}

function loadData() {
    const xhr = new XMLHttpRequest();
    xhr.open('GET', '/api/v2/performance', true);

    xhr.onload = function() {
        if (xhr.status >= 200 && xhr.status < 300) {
            const data = JSON.parse(xhr.responseText);
            updateChart(data); // Обновляем график с новыми данными
        } else {
            console.error('The request failed!');
        }
    };

    xhr.onerror = function() {
        console.error('There was a problem with the request.');
    };

    xhr.send();
}

function startUpdating() {
    if (!updateInterval) {
        loadData(); // Load data immediately
        updateInterval = setInterval(loadData, updateFrequency);
    }
}

function stopUpdating() {
    clearInterval(updateInterval);
    updateInterval = null;
}

document.addEventListener('DOMContentLoaded', () => {
    chart = Highcharts.chart('performance', chartOptions);
    document.getElementById('startButton').addEventListener('click', startUpdating);
    document.getElementById('stopButton').addEventListener('click', stopUpdating);

    // Start updating by default
    startUpdating();
});