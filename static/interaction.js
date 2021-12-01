document.getElementById("age_range").addEventListener("input", updateVal);

function updateVal(){
	document.getElementById("selected_age").value = document.getElementById("age_range").value;
}
