$(function() {
  function parseOfx(ofxString) {
    var parser = new DOMParser();
    var doc = parser.parseFromString(ofxString, "text/xml");
    var stmttrn_list = doc
      .getElementsByTagName("OFX")[0]
      .getElementsByTagName("BANKMSGSRSV1")[0]
      .getElementsByTagName("STMTTRNRS")[0]
      .getElementsByTagName("STMTRS")[0]
      .getElementsByTagName("BANKTRANLIST")[0].childNodes;
    var ans = [];
    stmttrn_list.forEach(function(stmttrn) {
      var dtposted = stmttrn.getElementsByTagName("DTPOSTED")[0].childNodes[0]
        .nodeValue;
      var trnamt = stmttrn.getElementsByTagName("TRNAMT")[0].childNodes[0]
        .nodeValue;
      var memo = stmttrn.getElementsByTagName("MEMO")[0].childNodes[0]
        .nodeValue;
      ans.push({
        Date: dtposted.replace(/^(\d{4})(\d{2})(\d{2})$/, "$1-$2-$3"),
        Cents: Math.abs(parseInt(trnamt.replace(".", ""))),
        Description: memo
      });
    });
    return ans;
  }

  function addToLookupTable(lookup, bills, prefix) {
    bills.forEach(function(bill) {
      var key = bill.Date + ":" + bill.Cents;
      bill.Prefix = prefix;
      lookup[key] = lookup[key] || [];
      lookup[key].push(bill);
    });
  }

  function compareExternalBills(appBills, extBills) {
    var lookup = {};
    addToLookupTable(lookup, appBills, "MASSIKONE");
    addToLookupTable(lookup, extBills, "PANKKI");
    $("#entries").empty();
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
        $("#entries")
          .addClass("table table-bordered")
          .append(
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

  function initCompare(ofxString) {
    $.get({
      url: "/api/compare"
    })
      .done(function(appBills) {
        compareExternalBills(appBills, parseOfx(ofxString));
      })
      .fail(function(jqXHR) {
        alert("Error: " + jqXHR.statusText);
      });
  }

  function initComparePankkiparseri(entries) {
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
        compareExternalBills(appBills, extBills);
      })
      .fail(function(jqXHR) {
        alert("Error: " + jqXHR.statusText);
      });
  }

  function handleFiles(files) {
    var reader = new FileReader();
    reader.onload = function(e) {
      initCompare(e.target.result);
    };
    reader.readAsText(files[0]);
  }

  $("#hiddenFileInput").change(function() {
    handleFiles(this.files);
  });

  $("#fileUploadButton").click(function(e) {
    $("#hiddenFileInput").click();
    e.preventDefault(); // prevent navigation to "#"
  });

  Pankkiparseri.addToForm("compare-form", initComparePankkiparseri);
});
