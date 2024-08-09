document.addEventListener('DOMContentLoaded', async ()  =>{
    const url = "/api/v2/graphsWork";
    const j = await (await fetch(url)).json();

    let _proxy_all_ = [];
    let _proxy_work_ = [];
    let _time_ = [];

    for (let c of j) {
        let b = c.dateTime*1000
        // _time_.push(b)
        _proxy_all_.push([b, c.total])
        _proxy_work_.push([b, c.work])
    }

    Highcharts.chart('proxyInfo', {
        chart: {
            zoomType: 'x',
            panning: true,
            panKey: 'shift',
            height: 360,
            type: "spline"
        },
        title: {
            text: 'Количество прокси'
        },

        subtitle: {
            text: ''
        },

        yAxis: {
            title: {
                text: 'Количество'
            },
            lineWidth: 1
        },
        xAxis: {
            title: {
                text: 'Время'
            },
            type: 'datetime',
            dateTimeLabelFormats: {
                month: '%e. %b',
            },
            crosshair: true,
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
            name: 'Рабочие прокси',
            data: _proxy_work_
        }, {
            name: 'Все прокси',
            data: _proxy_all_
        }
        ],

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

    });
})


