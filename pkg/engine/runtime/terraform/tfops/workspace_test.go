package tfops

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"bou.ke/monkey"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/spf13/afero"
	"github.com/zclconf/go-cty/cty"

	"kusionstack.io/kusion/pkg/engine/models"
)

var (
	resourceTest = models.Resource{
		ID:   "hashicorp:local:local_file:kusion_example",
		Type: "Terraform",
		Attributes: map[string]interface{}{
			"content":  "kusion",
			"filename": "test.txt",
		},
		Extensions: map[string]interface{}{
			"provider":     "registry.terraform.io/hashicorp/local/2.2.3",
			"resourceType": "local_file",
		},
	}
	tfstateTest = TFState{
		FormatVersion:    "0.2",
		TerraformVersion: "1.0.6",
		Values: &stateValues{
			RootModule: module{
				Resources: []resource{
					{
						Address:       "local_file.kusion_example",
						Mode:          "managed",
						Type:          "local_file",
						Name:          "kusion_example",
						ProviderName:  "registry.terraform.io/hashicorp/local",
						SchemaVersion: 0,
						AttributeValues: attributeValues{
							"content":              "kusion",
							"directory_permission": "0777",
							"file_permission":      "0777",
							"filename":             "text.txt",
							"sensitive_content":    nil,
							"source":               nil,
							"content_base64":       nil,
						},
					},
				},
			},
		},
	}

	fs = afero.Afero{Fs: afero.NewOsFs()}
)

func TestWriteHCL(t *testing.T) {
	type args struct {
		w *WorkSpace
	}

	type want struct {
		maintf string
	}

	cases := map[string]struct {
		args
		want
	}{
		"writeSuccess": {
			args: args{
				w: NewWorkSpace(fs),
			},
			want: want{
				maintf: "{\n  \"provider\": {\n    \"local\": null\n  },\n  \"resource\": {\n    \"local_file\": {\n      \"kusion_example\": {\n        \"content\": \"kusion\",\n        \"filename\": \"test.txt\"\n      }\n    }\n  },\n  \"terraform\": {\n    \"required_providers\": {\n      \"local\": {\n        \"source\": \"registry.terraform.io/hashicorp/local\",\n        \"version\": \"2.2.3\"\n      }\n    }\n  }\n}",
			},
		},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			tt.args.w.SetResource(&resourceTest)
			tt.args.w.SetCacheDir(".test")
			if err := tt.args.w.WriteHCL(); err != nil {
				t.Errorf("writeHCL error: %v", err)
			}

			s, _ := fs.ReadFile(filepath.Join(tt.w.tfCacheDir, "main.tf.json"))
			if diff := cmp.Diff(string(s), tt.want.maintf); diff != "" {
				t.Errorf("\n%s\nWriteHCL(...): -want maintf, +got maintf:\n%s", name, diff)
			}
		})
	}
}

func TestWriteTFState(t *testing.T) {
	type args struct {
		w *WorkSpace
	}

	type want struct {
		tfstate string
	}

	cases := map[string]struct {
		args
		want
	}{
		"writeSuccess": {
			args: args{
				w: NewWorkSpace(fs),
			},
			want: want{
				tfstate: "{\n  \"resources\": [\n    {\n      \"instances\": [\n        {\n          \"attributes\": {\n            \"content\": \"kusion\",\n            \"filename\": \"test.txt\"\n          }\n        }\n      ],\n      \"mode\": \"managed\",\n      \"name\": \"kusion_example\",\n      \"provider\": \"provider[\\\"registry.terraform.io/hashicorp/local\\\"]\",\n      \"type\": \"local_file\"\n    }\n  ],\n  \"version\": 4\n}",
			},
		},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			tt.args.w.SetResource(&resourceTest)
			tt.args.w.SetCacheDir(".test")
			if err := tt.args.w.WriteTFState(&resourceTest); err != nil {
				t.Errorf("WriteTFState error: %v", err)
			}

			s, _ := fs.ReadFile(filepath.Join(tt.w.tfCacheDir, "terraform.tfstate"))
			if diff := cmp.Diff(string(s), tt.want.tfstate); diff != "" {
				t.Errorf("\n%s\nWriteTFState(...): -want tfstate, +got tfstate:\n%s", name, diff)
			}
		})
	}
}

func TestInitWorkspace(t *testing.T) {
	type args struct {
		w *WorkSpace
	}

	type want struct {
		err error
	}

	cases := map[string]struct {
		args
		want
	}{
		"initws": {
			args: args{
				w: NewWorkSpace(fs),
			},
		},
	}
	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			tt.args.w.SetResource(&resourceTest)
			tt.args.w.SetCacheDir(".test")
			err := tt.args.w.InitWorkSpace(context.TODO())
			if diff := cmp.Diff(tt.want.err, err); diff != "" {
				t.Errorf("\nInitWorkSpace(...) -want err, +got err: \n%s", diff)
			}
		})
	}
}

func TestApply(t *testing.T) {
	type args struct {
		w *WorkSpace
	}

	tests := map[string]struct {
		args
	}{
		"applySuccess": {
			args: args{
				w: NewWorkSpace(fs),
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tt.w.SetResource(&resourceTest)
			tt.w.SetCacheDir(".test")
			tt.args.w.SetStackDir(".")
			if err := tt.w.WriteHCL(); err != nil {
				t.Errorf("\nWriteHCL error: %v", err)
			}
			if err := tt.w.InitWorkSpace(context.TODO()); err != nil {
				t.Errorf("\nInitWorkSpace error: %v", err)
			}
			if _, err := tt.w.Apply(context.TODO()); err != nil {
				t.Errorf("\n Apply error: %v", err)
			}
		})
	}
}

func TestRead(t *testing.T) {
	type args struct {
		w *WorkSpace
	}
	tests := map[string]struct {
		args args
	}{
		"readSuccess": {
			args: args{
				w: NewWorkSpace(fs),
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tt.args.w.SetResource(&resourceTest)
			tt.args.w.SetCacheDir(".test")
			tt.args.w.SetStackDir(".")
			if _, err := tt.args.w.Read(context.TODO()); err != nil {
				t.Errorf("\n Read error: %v", err)
			}
		})
	}
}

func TestRefreshOnly(t *testing.T) {
	type args struct {
		w *WorkSpace
	}
	tests := map[string]struct {
		args args
	}{
		"readSuccess": {
			args: args{
				w: NewWorkSpace(fs),
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tt.args.w.SetResource(&resourceTest)
			tt.args.w.SetCacheDir(".test")
			tt.args.w.SetStackDir(".")
			if _, err := tt.args.w.RefreshOnly(context.TODO()); err != nil {
				t.Errorf("\n RefreshOnly error: %v", err)
			}
		})
	}
}

func TestGerProvider(t *testing.T) {
	defer monkey.UnpatchAll()
	type args struct {
		w *WorkSpace
	}

	type want struct {
		addr string
		err  error
	}

	tests := map[string]struct {
		args
		want
	}{
		"Success": {
			args: args{
				w: NewWorkSpace(fs),
			},
			want: want{
				addr: "registry.terraform.io/hashicorp/local/2.2.3",
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockProviderAddr()
			addr, err := tt.args.w.GetProvider()
			if diff := cmp.Diff(tt.want.addr, addr); diff != "" {
				t.Errorf("\nGetProvider(...) -want addr, +got addr:\n%s", diff)
			}
			if diff := cmp.Diff(tt.want.err, err); diff != "" {
				t.Errorf("\nGetProvider(...) -want error, +got error:\n%s", diff)
			}
		})
	}
}

func mockProviderAddr() {
	monkey.Patch((*hclparse.Parser).ParseHCLFile, func(parse *hclparse.Parser, fileName string) (*hcl.File, hcl.Diagnostics) {
		return &hcl.File{
			Body: &hclsyntax.Body{
				Blocks: []*hclsyntax.Block{
					{
						Type:   "provider",
						Labels: []string{"registry.terraform.io/hashicorp/local"},
						Body: &hclsyntax.Body{
							Attributes: hclsyntax.Attributes{
								"version": &hclsyntax.Attribute{
									Name: "version",
									Expr: &hclsyntax.TemplateExpr{
										Parts: []hclsyntax.Expression{
											&hclsyntax.LiteralValueExpr{
												Val: cty.StringVal("2.2.3"),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}, nil
	})
}

func TestDestory(t *testing.T) {
	defer os.RemoveAll(".test")
	type args struct {
		w *WorkSpace
	}

	type want struct {
		err error
	}

	cases := map[string]struct {
		args
		want
	}{
		"success": {
			args: args{
				w: NewWorkSpace(fs),
			},
			want: want{
				err: nil,
			},
		},
	}
	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			tt.args.w.SetResource(&resourceTest)
			tt.args.w.SetCacheDir(".test")
			tt.args.w.SetStackDir(".")
			if err := tt.w.Destroy(context.TODO()); err != nil {
				t.Errorf("terraform destroy error: %v", err)
			}
		})
	}
}
