let gridApi;
let Data;
let countPackage;
let totalAll;
let totalUniqueAll;
let totalWork;


const gridOptions = {
    columnDefs: [
        { field: "id", cellRenderer: params => `<a href="${params.data.url}">${params.data.id}</a>`},
        { field: "name", cellRenderer: params => `<a href="/package?id=${params.data.id}">${params.data.name}</a>`},
        { field: "total"},
        { field: "unique",sort: 'desc'},
        { field: "inCheck"},
        { field: "work" }
    ],
    defaultColDef: {
        flex: 1,
        resizable: true,
        sortable: true,
        sortingOrder: ["desc", "asc", null]// Включаем сортировку по умолчанию для всех столбцов
    },
    onGridReady: params => {
        gridApi = params.api;
    }
};

function updateInfo(json) {
    countPackage.textContent = json.countPackage;
    totalAll.textContent = json.totalAll;
    totalUniqueAll.textContent = json.totalUniqueAll;
    totalWork.textContent = json.totalWork;
}

function refresh() {
    const url = "/api/v2/stats";
    fetch(url, { cache: 'default' })
        .then(response => response.json())
        .then(data => {
            const scrollPosition = gridApi.getVerticalPixelRange().top;
            Data = data;
            gridApi.setGridOption('rowData', data.packages);
            updateInfo(data);
            gridApi.ensureIndexVisible(Math.floor(scrollPosition / gridApi.getSizesForCurrentTheme().rowHeight), "top"); // Восстанавливаем позицию прокрутки
        })
        .catch(error => console.error('Error fetching data:', error));
}

document.addEventListener('DOMContentLoaded', async () => {
    const gridDiv = document.querySelector("#packageTable");
    gridApi = agGrid.createGrid(gridDiv, gridOptions);
    countPackage = document.querySelector('#count-package');
    totalAll = document.querySelector('#total-all');
    totalUniqueAll = document.querySelector('#total-unique-all');
    totalWork = document.querySelector('#total-work');

    await refresh();
    setInterval(refresh, 5000);
});
