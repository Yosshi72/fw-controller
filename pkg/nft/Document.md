# package nftの説明
## chain.go
- nftablesに追加したいchainのテンプレを管理
- chainの追加・削除をしたいときはこのファイルを変更する
- InitChain関数： ChainListに保管されているchainをtablenに追加する

## create_rule.go
- rule.goにあるテンプレをもとに実際にruleを作成する

## nft.go
- fwReconcilerを動かすためにnftablesの値を取得・更新する
- 現在はprefixAddr, ct_state, interfaceの情報にのみ対応

## rule.go
- nftablesに追加したいruleのテンプレを管理

## table.go
- nftablesに追加したいtableのテンプレを管理

## wrapper.gp
- nft.goを動かすためにparseなど便利ツールを管理