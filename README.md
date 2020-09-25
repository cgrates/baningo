# baningo [![Build Status](https://secure.travis-ci.org/cgrates/baningo.png)](http://travis-ci.org/cgrates/baningo)
Go client for apiban.org

## Installation ##

`go get github.com/cgrates/baningo`

## Support ##
Join [CGRateS](http://www.cgrates.org/ "CGRateS Website") on Google Groups [here](https://groups.google.com/forum/#!forum/cgrates "CGRateS on GoogleGroups").

## License ##
baningo is released under the [MIT License](http://www.opensource.org/licenses/mit-license.php "MIT License").
Copyright (C) ITsysCOM GmbH. All Rights Reserved.

## Sample usage code ##
```
package main

import (
	"fmt"

	"github.com/cgrates/baningo"
)

func main() {
	apikeys := []string{"APIKey1", "APIKey2"}

	// get all banned IPs
	bannedIps, err := baningo.GetBannedIPs(apikeys...)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(bannedIps)

	// check only one IP
	isBanned, err := baningo.CheckIP("127.168.56.203", apikeys...)
	if err != nil {
		fmt.Println(err)
		return
	}
	if isBanned {
		fmt.Println("The IP is banned")
	} else {
		fmt.Println("The IP is not banned")
	}
}
```



