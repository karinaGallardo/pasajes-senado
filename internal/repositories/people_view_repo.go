package repositories

import (
	"context"
	"errors"
	"sistema-pasajes/internal/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type PeopleViewRepository struct {
	db  *mongo.Database
	ctx context.Context
}

func NewPeopleViewRepository(db *mongo.Database) *PeopleViewRepository {
	return &PeopleViewRepository{
		db:  db,
		ctx: context.Background(),
	}
}

func (r *PeopleViewRepository) WithContext(ctx context.Context) *PeopleViewRepository {
	return &PeopleViewRepository{
		db:  r.db,
		ctx: ctx,
	}
}

func (r *PeopleViewRepository) FindSenatorDataByCI(ci string) (*models.MongoPersonaView, error) {
	if r.db == nil {
		return nil, errors.New("conexión a MongoDB RRHH no establecida")
	}

	collection := r.db.Collection("view_people_pasajes")
	var ctx context.Context
	var cancel context.CancelFunc

	if r.ctx != nil && r.ctx != context.Background() {
		ctx = r.ctx
		cancel = func() {}
	} else {
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	}
	defer cancel()

	var result models.MongoPersonaView
	filter := bson.M{"ci": ci}

	if err := collection.FindOne(ctx, filter).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *PeopleViewRepository) FindAllActiveSenators() ([]models.MongoPersonaView, error) {
	if r.db == nil {
		return nil, errors.New("conexión a MongoDB RRHH no establecida")
	}

	collection := r.db.Collection("view_people_pasajes")
	var ctx context.Context
	var cancel context.CancelFunc

	if r.ctx != nil && r.ctx != context.Background() {
		ctx = r.ctx
		cancel = func() {}
	} else {
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	}
	defer cancel()

	filter := bson.M{
		"senador_data.active": true,
		"tipo_funcionario":    primitive.Regex{Pattern: "SENADOR_", Options: "i"},
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []models.MongoPersonaView
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}

func (r *PeopleViewRepository) FindAllActiveStaff() ([]models.MongoPersonaView, error) {
	if r.db == nil {
		return nil, errors.New("conexión a MongoDB RRHH no establecida")
	}

	collection := r.db.Collection("view_people_pasajes")
	var ctx context.Context
	var cancel context.CancelFunc

	if r.ctx != nil && r.ctx != context.Background() {
		ctx = r.ctx
		cancel = func() {}
	} else {
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	}
	defer cancel()

	filter := bson.M{
		"tipo_funcionario":    bson.M{"$in": []string{"FUNCIONARIO_PERMANENTE", "FUNCIONARIO_EVENTUAL"}},
		"senador_data.active": bson.M{"$ne": true},
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []models.MongoPersonaView
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}

func (r *PeopleViewRepository) SyncView(ctx context.Context) error {
	if r.db == nil {
		return errors.New("conexión a MongoDB RRHH no establecida")
	}

	pipeline := mongo.Pipeline{
		// 1. Funcionario Permanente Lookup
		primitive.D{
			primitive.E{
				Key: "$lookup",
				Value: primitive.D{
					primitive.E{Key: "from", Value: "permanents"},
					primitive.E{Key: "let", Value: primitive.D{primitive.E{Key: "permId", Value: "$permanent"}}},
					primitive.E{
						Key: "pipeline",
						Value: mongo.Pipeline{
							primitive.D{
								primitive.E{
									Key: "$match",
									Value: primitive.D{
										primitive.E{
											Key: "$expr",
											Value: primitive.D{
												primitive.E{
													Key: "$and",
													Value: bson.A{
														primitive.D{primitive.E{Key: "$eq", Value: bson.A{"$_id", "$$permId"}}},
														primitive.D{primitive.E{Key: "$eq", Value: bson.A{"$active", true}}},
														primitive.D{primitive.E{Key: "$eq", Value: bson.A{primitive.D{primitive.E{Key: "$type", Value: "$disconnection"}}, "missing"}}},
													},
												},
											},
										},
									},
								},
							},
							primitive.D{
								primitive.E{
									Key: "$lookup",
									Value: primitive.D{
										primitive.E{Key: "from", Value: "items"},
										primitive.E{Key: "localField", Value: "item"},
										primitive.E{Key: "foreignField", Value: "_id"},
										primitive.E{Key: "as", Value: "item_data"},
									},
								},
							},
							primitive.D{
								primitive.E{
									Key: "$unwind",
									Value: primitive.D{
										primitive.E{Key: "path", Value: "$item_data"},
										primitive.E{Key: "preserveNullAndEmptyArrays", Value: true},
									},
								},
							},
							primitive.D{
								primitive.E{
									Key: "$project",
									Value: primitive.D{
										primitive.E{Key: "active", Value: 1},
										primitive.E{Key: "disconnection", Value: 1},
										primitive.E{Key: "item_data.unit", Value: 1},
										primitive.E{Key: "item_data.denomination", Value: 1},
										primitive.E{Key: "item_data.active", Value: 1},
										primitive.E{Key: "item_data.dependence", Value: 1},
										primitive.E{Key: "item_data.position", Value: 1},
									},
								},
							},
						},
					},
					primitive.E{Key: "as", Value: "funcionario_permanente"},
				},
			},
		},

		// 2. Funcionario Eventual Lookup
		primitive.D{
			primitive.E{
				Key: "$lookup",
				Value: primitive.D{
					primitive.E{Key: "from", Value: "eventuals"},
					primitive.E{Key: "let", Value: primitive.D{primitive.E{Key: "eventId", Value: "$eventual"}}},
					primitive.E{
						Key: "pipeline",
						Value: mongo.Pipeline{
							primitive.D{
								primitive.E{
									Key: "$match",
									Value: primitive.D{
										primitive.E{
											Key: "$expr",
											Value: primitive.D{
												primitive.E{Key: "$eq", Value: bson.A{"$_id", "$$eventId"}},
											},
										},
									},
								},
							},
							primitive.D{
								primitive.E{
									Key: "$lookup",
									Value: primitive.D{
										primitive.E{Key: "from", Value: "eventual_units"},
										primitive.E{Key: "localField", Value: "unit"},
										primitive.E{Key: "foreignField", Value: "_id"},
										primitive.E{Key: "as", Value: "unit_data"},
									},
								},
							},
							primitive.D{
								primitive.E{
									Key: "$unwind",
									Value: primitive.D{
										primitive.E{Key: "path", Value: "$unit_data"},
										primitive.E{Key: "preserveNullAndEmptyArrays", Value: true},
									},
								},
							},
							primitive.D{
								primitive.E{
									Key: "$lookup",
									Value: primitive.D{
										primitive.E{Key: "from", Value: "eventual_scales"},
										primitive.E{Key: "localField", Value: "scale"},
										primitive.E{Key: "foreignField", Value: "_id"},
										primitive.E{Key: "as", Value: "scale_data"},
									},
								},
							},
							primitive.D{
								primitive.E{
									Key: "$unwind",
									Value: primitive.D{
										primitive.E{Key: "path", Value: "$scale_data"},
										primitive.E{Key: "preserveNullAndEmptyArrays", Value: true},
									},
								},
							},
							primitive.D{
								primitive.E{
									Key: "$project",
									Value: primitive.D{
										primitive.E{Key: "endDate", Value: 1},
										primitive.E{Key: "unit_data.unit", Value: 1},
										primitive.E{Key: "unit_data.name", Value: 1},
										primitive.E{Key: "unit_data.description", Value: 1},
										primitive.E{Key: "unit_data.management", Value: 1},
										primitive.E{Key: "scale_data.position", Value: 1},
									},
								},
							},
						},
					},
					primitive.E{Key: "as", Value: "funcionario_eventual"},
				},
			},
		},

		// 3. Filtrar los que tienen algún contrato
		primitive.D{
			primitive.E{
				Key: "$match",
				Value: primitive.D{
					primitive.E{
						Key: "$or",
						Value: bson.A{
							primitive.D{primitive.E{Key: "funcionario_permanente", Value: primitive.D{primitive.E{Key: "$ne", Value: bson.A{}}}}},
							primitive.D{primitive.E{Key: "funcionario_eventual", Value: primitive.D{primitive.E{Key: "$ne", Value: bson.A{}}}}},
						},
					},
				},
			},
		},

		// 4. Aplanar campos de contrato
		primitive.D{
			primitive.E{
				Key: "$addFields",
				Value: primitive.D{
					primitive.E{
						Key:   "funcionario_permanente",
						Value: primitive.D{primitive.E{Key: "$arrayElemAt", Value: bson.A{"$funcionario_permanente", 0}}},
					},
					primitive.E{
						Key:   "funcionario_eventual",
						Value: primitive.D{primitive.E{Key: "$arrayElemAt", Value: bson.A{"$funcionario_eventual", 0}}},
					},
				},
			},
		},

		// 5. Lookup Senadores
		primitive.D{
			primitive.E{
				Key: "$lookup",
				Value: primitive.D{
					primitive.E{Key: "from", Value: "senadores"},
					primitive.E{Key: "localField", Value: "ci"},
					primitive.E{Key: "foreignField", Value: "ci"},
					primitive.E{Key: "as", Value: "datos_senador"},
				},
			},
		},
		primitive.D{
			primitive.E{
				Key: "$addFields",
				Value: primitive.D{
					primitive.E{
						Key:   "datos_senador",
						Value: primitive.D{primitive.E{Key: "$arrayElemAt", Value: bson.A{"$datos_senador", 0}}},
					},
				},
			},
		},

		// 6. Lookup Titular Relacionado
		primitive.D{
			primitive.E{
				Key: "$lookup",
				Value: primitive.D{
					primitive.E{Key: "from", Value: "senadores"},
					primitive.E{Key: "let", Value: primitive.D{primitive.E{Key: "my_ci", Value: "$ci"}}},
					primitive.E{
						Key: "pipeline",
						Value: mongo.Pipeline{
							primitive.D{
								primitive.E{
									Key: "$match",
									Value: primitive.D{
										primitive.E{
											Key: "$expr",
											Value: primitive.D{
												primitive.E{
													Key: "$and",
													Value: bson.A{
														primitive.D{primitive.E{Key: "$eq", Value: bson.A{"$suplente", "$$my_ci"}}},
														primitive.D{primitive.E{Key: "$ne", Value: bson.A{"$suplente", nil}}},
														primitive.D{primitive.E{Key: "$ne", Value: bson.A{"$suplente", ""}}},
													},
												},
											},
										},
									},
								},
							},
							primitive.D{
								primitive.E{
									Key: "$project",
									Value: primitive.D{
										primitive.E{Key: "ci", Value: 1},
									},
								},
							},
						},
					},
					primitive.E{Key: "as", Value: "titular_relacionado"},
				},
			},
		},
		primitive.D{
			primitive.E{
				Key: "$addFields",
				Value: primitive.D{
					primitive.E{
						Key:   "titular_relacionado",
						Value: primitive.D{primitive.E{Key: "$arrayElemAt", Value: bson.A{"$titular_relacionado", 0}}},
					},
				},
			},
		},

		// 7. Proyección Final
		primitive.D{
			primitive.E{
				Key: "$project",
				Value: primitive.D{
					primitive.E{Key: "_id", Value: 1},
					primitive.E{Key: "firstname", Value: 1},
					primitive.E{Key: "secondname", Value: 1},
					primitive.E{Key: "lastname", Value: 1},
					primitive.E{Key: "surname", Value: 1},
					primitive.E{Key: "ci", Value: 1},
					primitive.E{Key: "cie", Value: 1},
					primitive.E{Key: "nacdate", Value: 1},
					primitive.E{Key: "nationality", Value: 1},
					primitive.E{Key: "email", Value: 1},
					primitive.E{Key: "phone", Value: 1},
					primitive.E{Key: "address", Value: 1},
					primitive.E{Key: "nua", Value: 1},
					primitive.E{Key: "account", Value: 1},
					primitive.E{Key: "gender", Value: 1},
					primitive.E{Key: "biometricId", Value: 1},
					primitive.E{Key: "photo", Value: 1},
					primitive.E{
						Key: "senador_data",
						Value: primitive.D{
							primitive.E{
								Key: "$cond",
								Value: primitive.D{
									primitive.E{
										Key: "if",
										Value: primitive.D{
											primitive.E{
												Key: "$gt",
												Value: bson.A{
													primitive.D{primitive.E{Key: "$type", Value: "$datos_senador"}},
													"missing",
												},
											},
										},
									},
									primitive.E{
										Key: "then",
										Value: primitive.D{
											primitive.E{Key: "departamento", Value: "$datos_senador.departamento"},
											primitive.E{Key: "sigla", Value: "$datos_senador.sigla"},
											primitive.E{
												Key: "tipo",
												Value: primitive.D{
													primitive.E{
														Key: "$switch",
														Value: primitive.D{
															primitive.E{
																Key: "branches",
																Value: bson.A{
																	primitive.D{
																		primitive.E{
																			Key: "case",
																			Value: primitive.D{
																				primitive.E{
																					Key: "$and",
																					Value: bson.A{
																						primitive.D{primitive.E{Key: "$gt", Value: bson.A{"$titular_relacionado.ci", nil}}},
																						primitive.D{primitive.E{Key: "$ne", Value: bson.A{"$titular_relacionado.ci", ""}}},
																					},
																				},
																			},
																		},
																		primitive.E{Key: "then", Value: "SUPLENTE"},
																	},
																	primitive.D{
																		primitive.E{
																			Key: "case",
																			Value: primitive.D{
																				primitive.E{
																					Key: "$and",
																					Value: bson.A{
																						primitive.D{primitive.E{Key: "$gt", Value: bson.A{"$datos_senador.suplente", nil}}},
																						primitive.D{primitive.E{Key: "$ne", Value: bson.A{"$datos_senador.suplente", ""}}},
																					},
																				},
																			},
																		},
																		primitive.E{Key: "then", Value: "TITULAR"},
																	},
																	primitive.D{
																		primitive.E{
																			Key: "case",
																			Value: primitive.D{
																				primitive.E{Key: "$gt", Value: bson.A{"$datos_senador.tipo", nil}},
																			},
																		},
																		primitive.E{Key: "then", Value: "$datos_senador.tipo"},
																	},
																	primitive.D{
																		primitive.E{
																			Key: "case",
																			Value: primitive.D{
																				primitive.E{Key: "$gt", Value: bson.A{"$datos_senador.type", nil}},
																			},
																		},
																		primitive.E{Key: "then", Value: "$datos_senador.type"},
																	},
																},
															},
															primitive.E{Key: "default", Value: "TITULAR"},
														},
													},
												},
											},
											primitive.E{Key: "su_suplente_ci", Value: "$datos_senador.suplente"},
											primitive.E{
												Key: "su_titular_ci",
												Value: primitive.D{
													primitive.E{Key: "$ifNull", Value: bson.A{"$datos_senador.titular", "$titular_relacionado.ci"}},
												},
											},
											primitive.E{Key: "gestion", Value: "$datos_senador.gestion"},
											primitive.E{Key: "active", Value: "$datos_senador.active"},
										},
									},
									primitive.E{Key: "else", Value: "$$REMOVE"},
								},
							},
						},
					},
					primitive.E{
						Key: "dependencia",
						Value: primitive.D{
							primitive.E{
								Key: "$switch",
								Value: primitive.D{
									primitive.E{
										Key: "branches",
										Value: bson.A{
											primitive.D{
												primitive.E{
													Key: "case",
													Value: primitive.D{
														primitive.E{
															Key: "$gt",
															Value: bson.A{
																primitive.D{primitive.E{Key: "$type", Value: "$funcionario_eventual"}},
																"missing",
															},
														},
													},
												},
												primitive.E{Key: "then", Value: "$funcionario_eventual.unit_data.unit"},
											},
											primitive.D{
												primitive.E{
													Key: "case",
													Value: primitive.D{
														primitive.E{
															Key: "$and",
															Value: bson.A{
																primitive.D{primitive.E{Key: "$gt", Value: bson.A{primitive.D{primitive.E{Key: "$type", Value: "$funcionario_permanente"}}, "missing"}}},
																primitive.D{primitive.E{Key: "$eq", Value: bson.A{"$funcionario_permanente.active", true}}},
																primitive.D{primitive.E{Key: "$eq", Value: bson.A{primitive.D{primitive.E{Key: "$type", Value: "$funcionario_permanente.disconnection"}}, "missing"}}},
															},
														},
													},
												},
												primitive.E{Key: "then", Value: "$funcionario_permanente.item_data.unit"},
											},
										},
									},
									primitive.E{Key: "default", Value: "$$REMOVE"},
								},
							},
						},
					},
					primitive.E{
						Key: "cargo",
						Value: primitive.D{
							primitive.E{
								Key: "$switch",
								Value: primitive.D{
									primitive.E{
										Key: "branches",
										Value: bson.A{
											primitive.D{
												primitive.E{
													Key: "case",
													Value: primitive.D{
														primitive.E{
															Key: "$gt",
															Value: bson.A{
																primitive.D{primitive.E{Key: "$type", Value: "$funcionario_eventual"}},
																"missing",
															},
														},
													},
												},
												primitive.E{Key: "then", Value: "$funcionario_eventual.scale_data.position"},
											},
											primitive.D{
												primitive.E{
													Key: "case",
													Value: primitive.D{
														primitive.E{
															Key: "$and",
															Value: bson.A{
																primitive.D{primitive.E{Key: "$gt", Value: bson.A{primitive.D{primitive.E{Key: "$type", Value: "$funcionario_permanente"}}, "missing"}}},
																primitive.D{primitive.E{Key: "$eq", Value: bson.A{"$funcionario_permanente.active", true}}},
																primitive.D{primitive.E{Key: "$eq", Value: bson.A{primitive.D{primitive.E{Key: "$type", Value: "$funcionario_permanente.disconnection"}}, "missing"}}},
															},
														},
													},
												},
												primitive.E{Key: "then", Value: "$funcionario_permanente.item_data.denomination"},
											},
										},
									},
									primitive.E{Key: "default", Value: "$$REMOVE"},
								},
							},
						},
					},
					primitive.E{
						Key: "funcionario_permanente",
						Value: primitive.D{
							primitive.E{
								Key: "$cond",
								Value: primitive.D{
									primitive.E{
										Key: "if",
										Value: primitive.D{
											primitive.E{
												Key: "$and",
												Value: bson.A{
													primitive.D{primitive.E{Key: "$gt", Value: bson.A{primitive.D{primitive.E{Key: "$type", Value: "$funcionario_permanente"}}, "missing"}}},
													primitive.D{primitive.E{Key: "$eq", Value: bson.A{"$funcionario_permanente.active", true}}},
													primitive.D{primitive.E{Key: "$eq", Value: bson.A{primitive.D{primitive.E{Key: "$type", Value: "$funcionario_permanente.disconnection"}}, "missing"}}},
												},
											},
										},
									},
									primitive.E{
										Key: "then",
										Value: primitive.D{
											primitive.E{Key: "item_data", Value: "$funcionario_permanente.item_data"},
											primitive.E{Key: "active", Value: "$funcionario_permanente.active"},
											primitive.E{Key: "disconnection", Value: "$funcionario_permanente.disconnection"},
										},
									},
									primitive.E{Key: "else", Value: "$$REMOVE"},
								},
							},
						},
					},
					primitive.E{
						Key: "funcionario_eventual",
						Value: primitive.D{
							primitive.E{
								Key: "$cond",
								Value: primitive.D{
									primitive.E{
										Key: "if",
										Value: primitive.D{
											primitive.E{
												Key: "$gt",
												Value: bson.A{
													primitive.D{primitive.E{Key: "$type", Value: "$funcionario_eventual"}},
													"missing",
												},
											},
										},
									},
									primitive.E{
										Key: "then",
										Value: primitive.D{
											primitive.E{Key: "unit_data", Value: "$funcionario_eventual.unit_data"},
											primitive.E{Key: "scale_data", Value: "$funcionario_eventual.scale_data"},
											primitive.E{Key: "endDate", Value: "$funcionario_eventual.endDate"},
										},
									},
									primitive.E{Key: "else", Value: "$$REMOVE"},
								},
							},
						},
					},
					primitive.E{
						Key: "tipo_funcionario",
						Value: primitive.D{
							primitive.E{
								Key: "$switch",
								Value: primitive.D{
									primitive.E{
										Key: "branches",
										Value: bson.A{
											primitive.D{
												primitive.E{
													Key: "case",
													Value: primitive.D{
														primitive.E{
															Key: "$and",
															Value: bson.A{
																primitive.D{primitive.E{Key: "$gt", Value: bson.A{primitive.D{primitive.E{Key: "$type", Value: "$datos_senador"}}, "missing"}}},
																primitive.D{primitive.E{Key: "$gt", Value: bson.A{"$titular_relacionado.ci", nil}}},
															},
														},
													},
												},
												primitive.E{Key: "then", Value: "SENADOR_SUPLENTE"},
											},
											primitive.D{
												primitive.E{
													Key: "case",
													Value: primitive.D{
														primitive.E{
															Key: "$gt",
															Value: bson.A{
																primitive.D{primitive.E{Key: "$type", Value: "$datos_senador"}},
																"missing",
															},
														},
													},
												},
												primitive.E{Key: "then", Value: "SENADOR_TITULAR"},
											},
											primitive.D{
												primitive.E{
													Key: "case",
													Value: primitive.D{
														primitive.E{
															Key: "$gt",
															Value: bson.A{
																primitive.D{primitive.E{Key: "$type", Value: "$funcionario_eventual"}},
																"missing",
															},
														},
													},
												},
												primitive.E{Key: "then", Value: "FUNCIONARIO_EVENTUAL"},
											},
											primitive.D{
												primitive.E{
													Key: "case",
													Value: primitive.D{
														primitive.E{
															Key: "$and",
															Value: bson.A{
																primitive.D{primitive.E{Key: "$gt", Value: bson.A{primitive.D{primitive.E{Key: "$type", Value: "$funcionario_permanente"}}, "missing"}}},
																primitive.D{primitive.E{Key: "$eq", Value: bson.A{"$funcionario_permanente.active", true}}},
																primitive.D{primitive.E{Key: "$eq", Value: bson.A{primitive.D{primitive.E{Key: "$type", Value: "$funcionario_permanente.disconnection"}}, "missing"}}},
															},
														},
													},
												},
												primitive.E{Key: "then", Value: "FUNCIONARIO_PERMANENTE"},
											},
										},
									},
									primitive.E{Key: "default", Value: nil},
								},
							},
						},
					},
				},
			},
		},

		// 8. Filtrar finales válidos
		primitive.D{
			primitive.E{
				Key: "$match",
				Value: primitive.D{
					primitive.E{
						Key: "$and",
						Value: bson.A{
							primitive.D{
								primitive.E{
									Key: "$or",
									Value: bson.A{
										primitive.D{primitive.E{Key: "funcionario_permanente", Value: primitive.D{primitive.E{Key: "$exists", Value: true}}}},
										primitive.D{primitive.E{Key: "funcionario_eventual", Value: primitive.D{primitive.E{Key: "$exists", Value: true}}}},
									},
								},
							},
							primitive.D{
								primitive.E{
									Key: "$nor",
									Value: bson.A{
										primitive.D{
											primitive.E{
												Key: "$and",
												Value: bson.A{
													primitive.D{
														primitive.E{
															Key: "$or",
															Value: bson.A{
																primitive.D{primitive.E{Key: "tipo_funcionario", Value: "SENADOR_TITULAR"}},
																primitive.D{primitive.E{Key: "tipo_funcionario", Value: "SENADOR_SUPLENTE"}},
															},
														},
													},
													primitive.D{
														primitive.E{
															Key:   "senador_data",
															Value: primitive.D{primitive.E{Key: "$exists", Value: false}},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},

		// 9. Limpiar campos temporales y Salida
		primitive.D{
			primitive.E{
				Key: "$project",
				Value: primitive.D{
					primitive.E{Key: "funcionario_permanente", Value: 0},
					primitive.E{Key: "funcionario_eventual", Value: 0},
				},
			},
		},
		primitive.D{
			primitive.E{
				Key:   "$out",
				Value: "view_people_pasajes",
			},
		},
	}

	_, err := r.db.Collection("peoples").Aggregate(ctx, pipeline)
	return err
}
