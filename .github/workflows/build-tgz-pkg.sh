PKG_ROOT=/tmp
VERSION=$TAG
AXIA_ROOT=$PKG_ROOT/axia-$VERSION

mkdir -p $AXIA_ROOT

OK=`cp ./build/axia $AXIA_ROOT`
if [[ $OK -ne 0 ]]; then
  exit $OK;
fi
OK=`cp -r ./build/plugins $AXIA_ROOT`
if [[ $OK -ne 0 ]]; then
  exit $OK;
fi


echo "Build tgz package..."
cd $PKG_ROOT
echo "Version: $VERSION"
tar -czvf "axia-linux-$ARCH-$VERSION.tar.gz" axia-$VERSION
aws s3 cp axia-linux-$ARCH-$VERSION.tar.gz s3://$BUCKET/linux/binaries/ubuntu/$RELEASE/$ARCH/
rm -rf $PKG_ROOT/axia*
