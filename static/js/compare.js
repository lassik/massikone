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
        date: dtposted.replace(/^(\d{4})(\d{2})(\d{2})$/, "$1-$2-$3"),
        cents: Math.abs(parseInt(trnamt.replace(".", ""))),
        description: memo
      });
    });
    return ans;
  }

  function addToLookupTable(lookup, bills, prefix) {
    bills.forEach(function(bill) {
      var key = bill.date + ":" + bill.cents;
      bill.prefix = prefix;
      lookup[key] = lookup[key] || [];
      lookup[key].push(bill);
    });
  }

  function compare(ofxString, appBills) {
    var ofxBills = parseOfx(ofxString);
    var lookup = {};
    addToLookupTable(lookup, appBills, "MASSIKONE");
    addToLookupTable(lookup, ofxBills, "PANKKI");
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
              .append($("<td>").text(isFirst ? bill.date : ""))
              .append($("<td>").text(isFirst ? bill.cents : ""))
              .append($("<td>").text(bill.prefix))
              .append($("<td>").text(bill.description))
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
        compare(ofxString, appBills);
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
});
