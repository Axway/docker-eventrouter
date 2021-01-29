#!/bin/bash
#
# DO NOT ALTER OR REMOVE COPYRIGHT NOTICES OR THIS HEADER.
#
# Copyright (c) 2021 Axway Software SA and its affiliates. All rights reserved.
#

. $ER_INSTALLDIR/conf/profile

cmd="agtcmd status"
out=$($cmd)
rc=$?

exit $rc
