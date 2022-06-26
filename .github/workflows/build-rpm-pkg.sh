PKG_ROOT=/tmp/axia
RPM_BASE_DIR=$PKG_ROOT/yum
AXIA_BUILD_BIN_DIR=$RPM_BASE_DIR/usr/local/bin
AXIA_LIB_DIR=$RPM_BASE_DIR/usr/local/lib/axia

mkdir -p $RPM_BASE_DIR
mkdir -p $AXIA_BUILD_BIN_DIR
mkdir -p $AXIA_LIB_DIR

OK=`cp ./build/axia $AXIA_BUILD_BIN_DIR`
if [[ $OK -ne 0 ]]; then
  exit $OK;
fi
OK=`cp ./build/plugins/evm $AXIA_LIB_DIR`
if [[ $OK -ne 0 ]]; then
  exit $OK;
fi

echo "Build rpm package..."
VER=$(echo $TAG | gawk -F- '{print$1}' | tr -d 'v' )
REL=$(echo $TAG | gawk -F- '{print$2}')
[ -z "$REL" ] && REL=0 
echo "Tag: $VER"
rpmbuild --bb --define "version $VER" --define "release $REL" --buildroot $RPM_BASE_DIR .github/workflows/yum/specfile/axia.spec
aws s3 cp ~/rpmbuild/RPMS/x86_64/axia-*.rpm s3://$BUCKET/linux/rpm/
