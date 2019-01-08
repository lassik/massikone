$(function() {
  function addToLookupTable(lookup, bills, prefix) {
    bills.forEach(function(bill) {
      var key = bill.Date + ":" + bill.Cents;
      bill.Prefix = prefix;
      lookup[key] = lookup[key] || [];
      lookup[key].push(bill);
    });
  }

  function compare(appBills, extBills) {
    var lookup = {};
    addToLookupTable(lookup, appBills, "Massikone");
    addToLookupTable(lookup, extBills, "Pankki");
    $("#entries > tbody").empty();
    var lookupKeys = [];
    for (var key in lookup) {
      if (lookup.hasOwnProperty(key)) {
        lookupKeys.push(key);
      }
    }
    lookupKeys.sort();
    lookupKeys.forEach(function(key) {
      var keyBills = lookup[key];
      var isFirst = true;
      var cssClass = keyBills.length === 2 ? "success" : "danger";
      keyBills.forEach(function(bill) {
        $("#entries > tbody").append(
          $("<tr>")
            .addClass(cssClass)
            .append($("<td>").text(isFirst ? bill.Date : ""))
            .append($("<td>").text(isFirst ? bill.Cents : ""))
            .append($("<td>").text(bill.Prefix))
            .append($("<td>").text(bill.Description))
        );
        isFirst = false;
      });
    });
  }

  function initCompare(entries) {
    $.get({
      url: "/api/compare"
    })
      .done(function(appBills) {
        var extBills = [];
        for (var i = 0; i < entries.length; i++) {
          var entry = entries[i];
          extBills.push({
            Date: entry.date.finnish,
            Cents: entry.amount.cents,
            Description: entry.message
          });
        }
        compare(appBills, extBills);
      })
      .fail(function(jqXHR) {
        alert("Error: " + jqXHR.statusText);
      });
  }

  for (var i = Pankkiparseri.formatsList.length - 1; i >= 0; i--) {
    var format = Pankkiparseri.formatsList[i];
    $("<button>")
      .attr("class", "btn")
      .text(format.bankTitle)
      .click(
        Pankkiparseri.addBankToForm(
          document.getElementById("compare-form"),
          initCompare,
          format.parse,
          format.encoding
        )
      )
      .prependTo("#compare-btn-group");
  }
});
