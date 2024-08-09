let gridApi;
let Data;

const gridOptions = {
    dataTypeDefinitions: {
        ip: {
            extendsDataType: 'number',
            baseDataType: 'number',
            valueFormatter: params => `${[(params.value >>> 24) & 0xff, (params.value >>> 16) & 0xff, (params.value >>> 8) & 0xff, params.value & 0xff].join('.')}`,
        },
    },
    columnDefs: [
        { field: "packageId", filter: "agNumberColumnFilter"},
        { field: "proxy", filter: true},
        { field: "work", filter: "agBoolColumnFilter"},
        { field: "ipOut", cellDataType: "ip"},
        { field: "latency", filter: "agNumberColumnFilter"},
        { field: "dispersion" },
        { field: "percent", filter: "agNumberColumnFilter"},
        {
            field: "history",
            headerName: "History",
            cellRenderer: 'agSparklineCellRenderer',
            cellRendererParams: {
                sparklineOptions: {
                    type: 'line',
                },
                axes: [
                    {
                        type: 'number',
                        position: 'bottom',
                    },
                    {
                        type: 'number',
                        position: 'bottom',
                    },
                ],
            },
        },
        {field: "country", filter: "agBoolColumnFilter"},
        { field: "lastCheck", filter: "agDateColumnFilter", cellRenderer: (data) => {
                return moment(data.value).format('MM/DD/YYYY HH:mm:ss')
            }}
    ],
    defaultColDef: {
         filter: '',

    },
};


document.addEventListener("DOMContentLoaded", () => {
    const gridDiv = document.querySelector("#myGrid");
    gridApi = agGrid.createGrid(gridDiv, gridOptions);
    var params = window.location.search.replace('?','').split('&').reduce(
            function(p,e){
                var a = e.split('=');
                p[ decodeURIComponent(a[0])] = decodeURIComponent(a[1]);
                return p;
            },{});
    let url = '/api/v2/statsPackage';
    if (params['id']) {
        url = url.concat('?packageId=', params['id'])
    }
    fetch(url, { cache: 'default' })
        .then((response) => response.json()).then((data) => {
            Data = data
        data.proxyList = data.proxyList.map((item) => {
            history2 = item.history.map(value => value >= data.maxLatency ? undefined : value/1e3)
            item.history = history2
            // item.latency = count === 0 ? data.maxLatency : parseInt(sum / count / 1000);
            item.percent = parseFloat(Number((item.history.filter(x => x).length / item.history.length) * 1e2).toFixed(2))
            return item;
        });
            gridApi.setGridOption(
                "rowData", data.proxyList
            )

        }
    )
})



// function renderLatency(params) {

// }

function filterLatency(params) {
    console.log(params)
}