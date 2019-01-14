$(function() {
  function formatEuros(euros) {
    return euros.toFixed(2).replace(".", ",");
  }

  function formatCents(cents) {
    return formatEuros(cents / 100.0);
  }

  function addToLookupTable(lookup, documents, prefix) {
    documents.forEach(function(document) {
      var key = document.Date + ":" + document.Cents;
      document.Prefix = prefix;
      lookup[key] = lookup[key] || [];
      lookup[key].push(document);
    });
  }

  function compare(appDocuments, extDocuments) {
    var lookup = {};
    addToLookupTable(lookup, appDocuments, "Massikone");
    addToLookupTable(lookup, extDocuments, "Pankki");
    $("#entries > tbody").empty();
    var lookupKeys = [];
    for (var key in lookup) {
      if (lookup.hasOwnProperty(key)) {
        lookupKeys.push(key);
      }
    }
    lookupKeys.sort();
    lookupKeys.forEach(function(key) {
      var keyDocuments = lookup[key];
      var isFirst = true;
      var cssClass = keyDocuments.length === 2 ? "success" : "danger";
      keyDocuments.forEach(function(document) {
        $("#entries > tbody").append(
          $("<tr>")
            .addClass(cssClass)
            .append(
              $("<td>")
                .addClass("text-right")
                .text(isFirst ? document.Date : "")
            )
            .append(
              $("<td>")
                .addClass("text-right")
                .text(isFirst ? formatCents(document.Cents) : "")
            )
            .append($("<td>").text(document.Prefix))
            .append($("<td>").text(document.Description))
        );
        isFirst = false;
      });
    });
  }

  function initCompare(entries) {
    $.get({
      url: "/api/compare"
    })
      .done(function(appDocuments) {
        var extDocuments = [];
        for (var i = 0; i < entries.length; i++) {
          var entry = entries[i];
          extDocuments.push({
            Date: entry.date.finnish,
            Cents: entry.amount.cents,
            Description: entry.message
          });
        }
        compare(appDocuments, extDocuments);
      })
      .fail(function(jqXHR) {
        alert("Error: " + jqXHR.statusText);
      });
  }

  for (var i = 0; i < Pankkiparseri.formatsList.length; i++) {
    var format = Pankkiparseri.formatsList[i];
    $("#compare-btn-table").append(
      $("<tr>")
        .append(
          $("<td>").append(
            $("<button>")
              .addClass("btn btn-block btn-info")
              .text(format.bankTitle)
              .append("&hellip;")
              .click(
                Pankkiparseri.addBankToForm(
                  document.getElementById("compare-form"),
                  initCompare,
                  format.parse,
                  format.encoding
                )
              )
          )
        )
        .append($("<td>").text(format.subtitle))
    );
  }
});
